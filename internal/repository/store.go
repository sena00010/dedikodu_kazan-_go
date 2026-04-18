package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"dedikodu-kazani/backend/internal/auth"
	"dedikodu-kazani/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Store struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) UpsertFirebaseUser(ctx context.Context, identity auth.FirebaseIdentity) (models.User, error) {
	var user models.User
	err := s.db.GetContext(ctx, &user, `SELECT * FROM users WHERE firebase_uid = ? LIMIT 1`, identity.UID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return user, err
	}

	username := fmt.Sprintf("Dedikoducu%d", time.Now().UnixNano()%100000)
	fullName := strings.TrimSpace(identity.FullName)
	if fullName == "" {
		fullName = username
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO users (public_id, firebase_uid, email, username, full_name, avatar_url)
		VALUES (?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''))`,
		uuid.NewString(), identity.UID, identity.Email, username, fullName, identity.AvatarURL)
	if err != nil {
		return user, err
	}
	id, _ := res.LastInsertId()
	return s.GetUserByID(ctx, uint64(id))
}

func (s *Store) CreateEmailUser(ctx context.Context, email, passwordHash, fullName string) (models.User, error) {
	username := fmt.Sprintf("Dedikoducu%d", time.Now().UnixNano()%100000)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO users (public_id, email, password_hash, username, full_name)
		VALUES (?, ?, ?, ?, ?)`,
		uuid.NewString(), strings.ToLower(email), passwordHash, username, fullName)
	if err != nil {
		return models.User{}, err
	}
	id, _ := res.LastInsertId()
	return s.GetUserByID(ctx, uint64(id))
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (models.User, string, error) {
	var row struct {
		models.User
		PasswordHash string `db:"password_hash"`
	}
	err := s.db.GetContext(ctx, &row, `SELECT * FROM users WHERE email = ? LIMIT 1`, strings.ToLower(email))
	return row.User, row.PasswordHash, err
}

func (s *Store) GetUserByID(ctx context.Context, id uint64) (models.User, error) {
	var user models.User
	err := s.db.GetContext(ctx, &user, `SELECT * FROM users WHERE id = ? LIMIT 1`, id)
	return user, err
}

func (s *Store) UpdateProfile(ctx context.Context, userID uint64, fullName string, age *int, jobTitle, gender, partner, avatarURL, bio, languageCode *string) (models.User, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET full_name = ?, age = ?, job_title = ?, gender = ?, partner = ?,
			avatar_url = COALESCE(?, avatar_url), bio = ?, language_code = ?
		WHERE id = ?`, fullName, age, jobTitle, gender, partner, avatarURL, bio, languageCode, userID)
	if err != nil {
		return models.User{}, err
	}
	return s.GetUserByID(ctx, userID)
}

func (s *Store) ListThreads(ctx context.Context, limit, offset int) ([]models.Thread, error) {
	var threads []models.Thread
	err := s.db.SelectContext(ctx, &threads, `
		SELECT * FROM threads
		WHERE is_active = true
		ORDER BY last_activity_at DESC
		LIMIT ? OFFSET ?`, limit, offset)
	return threads, err
}

func (s *Store) CreateThread(ctx context.Context, userID uint64, content string) (models.Thread, error) {
	id := uuid.NewString()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO threads (id, author_id, content, last_activity_at)
		VALUES (?, ?, ?, NOW())`, id, userID, strings.TrimSpace(content))
	if err != nil {
		return models.Thread{}, err
	}
	var thread models.Thread
	err = s.db.GetContext(ctx, &thread, `SELECT * FROM threads WHERE id = ?`, id)
	return thread, err
}

func (s *Store) ListComments(ctx context.Context, threadID string, limit int) ([]models.Comment, error) {
	var comments []models.Comment
	err := s.db.SelectContext(ctx, &comments, `
		SELECT * FROM comments
		WHERE thread_id = ?
		ORDER BY created_at ASC
		LIMIT ?`, threadID, limit)
	return comments, err
}

func normalizeMessage(input models.MessageInput) models.MessageInput {
	input.Content = strings.TrimSpace(input.Content)
	if input.MessageType == "" {
		input.MessageType = "text"
	}
	return input
}

func (s *Store) CreateThreadComment(ctx context.Context, threadID string, userID *uint64, input models.MessageInput, isAI bool, persona *string) (models.Comment, error) {
	input = normalizeMessage(input)
	commentID := uuid.NewString()
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return models.Comment{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO comments (
			id, thread_id, user_id, is_ai, ai_persona_type, content, message_type,
			media_url, thumbnail_url, mime_type, file_name, file_size, duration_ms, reply_to_id
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		commentID, threadID, userID, isAI, persona, input.Content, input.MessageType,
		input.MediaURL, input.ThumbnailURL, input.MimeType, input.FileName, input.FileSize, input.DurationMS, input.ReplyToID)
	if err != nil {
		return models.Comment{}, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE threads SET comment_count = comment_count + 1, last_activity_at = NOW() WHERE id = ?`, threadID)
	if err != nil {
		return models.Comment{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.Comment{}, err
	}
	var comment models.Comment
	err = s.db.GetContext(ctx, &comment, `SELECT * FROM comments WHERE id = ?`, commentID)
	return comment, err
}

func (s *Store) CreateRoom(ctx context.Context, ownerID uint64, title string) (models.PrivateRoom, error) {
	id := uuid.NewString()
	code := strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", ""))
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO private_rooms (id, owner_id, title, invite_code)
		VALUES (?, ?, ?, ?)`, id, ownerID, strings.TrimSpace(title), code)
	if err != nil {
		return models.PrivateRoom{}, err
	}
	_, _ = s.db.ExecContext(ctx, `INSERT IGNORE INTO room_members (room_id, user_id, role) VALUES (?, ?, 'owner')`, id, ownerID)
	var room models.PrivateRoom
	err = s.db.GetContext(ctx, &room, `SELECT * FROM private_rooms WHERE id = ?`, id)
	return room, err
}

func (s *Store) JoinRoom(ctx context.Context, userID uint64, inviteCode string) (models.PrivateRoom, error) {
	var room models.PrivateRoom
	err := s.db.GetContext(ctx, &room, `SELECT * FROM private_rooms WHERE invite_code = ? LIMIT 1`, strings.ToUpper(inviteCode))
	if err != nil {
		return room, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT IGNORE INTO room_members (room_id, user_id, role) VALUES (?, ?, 'member')`, room.ID, userID)
	return room, err
}

func (s *Store) ListRoomComments(ctx context.Context, roomID string, limit int) ([]models.Comment, error) {
	var comments []models.Comment
	err := s.db.SelectContext(ctx, &comments, `
		SELECT * FROM comments
		WHERE room_id = ?
		ORDER BY created_at ASC
		LIMIT ?`, roomID, limit)
	return comments, err
}

func (s *Store) CreateRoomComment(ctx context.Context, roomID string, userID *uint64, input models.MessageInput, isAI bool, persona *string) (models.Comment, error) {
	input = normalizeMessage(input)
	commentID := uuid.NewString()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO comments (
			id, room_id, user_id, is_ai, ai_persona_type, content, message_type,
			media_url, thumbnail_url, mime_type, file_name, file_size, duration_ms, reply_to_id
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		commentID, roomID, userID, isAI, persona, input.Content, input.MessageType,
		input.MediaURL, input.ThumbnailURL, input.MimeType, input.FileName, input.FileSize, input.DurationMS, input.ReplyToID)
	if err != nil {
		return models.Comment{}, err
	}
	var comment models.Comment
	err = s.db.GetContext(ctx, &comment, `SELECT * FROM comments WHERE id = ?`, commentID)
	return comment, err
}

func (s *Store) GetOrCreateDirectConversation(ctx context.Context, userID, otherID uint64) (models.Conversation, error) {
	var conversation models.Conversation
	err := s.db.GetContext(ctx, &conversation, `
		SELECT c.* FROM conversations c
		JOIN conversation_members cm1 ON cm1.conversation_id = c.id AND cm1.user_id = ?
		JOIN conversation_members cm2 ON cm2.conversation_id = c.id AND cm2.user_id = ?
		WHERE c.conversation_type = 'direct'
		LIMIT 1`, userID, otherID)
	if err == nil {
		return conversation, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return conversation, err
	}
	id := uuid.NewString()
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return conversation, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO conversations (id, conversation_type) VALUES (?, 'direct')`, id); err != nil {
		return conversation, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?), (?, ?)`, id, userID, id, otherID); err != nil {
		return conversation, err
	}
	if err = tx.Commit(); err != nil {
		return conversation, err
	}
	err = s.db.GetContext(ctx, &conversation, `SELECT * FROM conversations WHERE id = ?`, id)
	return conversation, err
}

func (s *Store) ListConversationMessages(ctx context.Context, conversationID string, userID uint64, limit int) ([]models.Comment, error) {
	var allowed int
	if err := s.db.GetContext(ctx, &allowed, `SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ? AND user_id = ?`, conversationID, userID); err != nil || allowed == 0 {
		return nil, errors.New("conversation access denied")
	}
	var comments []models.Comment
	err := s.db.SelectContext(ctx, &comments, `
		SELECT * FROM comments
		WHERE conversation_id = ?
		ORDER BY created_at ASC
		LIMIT ?`, conversationID, limit)
	return comments, err
}

func (s *Store) CreateConversationMessage(ctx context.Context, conversationID string, userID uint64, input models.MessageInput) (models.Comment, error) {
	input = normalizeMessage(input)
	var allowed int
	if err := s.db.GetContext(ctx, &allowed, `SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ? AND user_id = ?`, conversationID, userID); err != nil || allowed == 0 {
		return models.Comment{}, errors.New("conversation access denied")
	}
	commentID := uuid.NewString()
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return models.Comment{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO comments (
			id, conversation_id, user_id, content, message_type,
			media_url, thumbnail_url, mime_type, file_name, file_size, duration_ms, reply_to_id
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		commentID, conversationID, userID, input.Content, input.MessageType,
		input.MediaURL, input.ThumbnailURL, input.MimeType, input.FileName, input.FileSize, input.DurationMS, input.ReplyToID)
	if err != nil {
		return models.Comment{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE conversations SET last_message_at = NOW() WHERE id = ?`, conversationID); err != nil {
		return models.Comment{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.Comment{}, err
	}
	var comment models.Comment
	err = s.db.GetContext(ctx, &comment, `SELECT * FROM comments WHERE id = ?`, commentID)
	return comment, err
}

func (s *Store) SaveUpload(ctx context.Context, ownerID uint64, url, fileName, mimeType, mediaType string, fileSize int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO uploads (id, owner_id, url, file_name, mime_type, file_size, media_type)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), ownerID, url, fileName, mimeType, fileSize, mediaType)
	return err
}

func (s *Store) IsRoomAIActive(ctx context.Context, roomID string) bool {
	var active bool
	err := s.db.GetContext(ctx, &active, `
		SELECT is_ai_active = true AND (ai_expires_at IS NULL OR ai_expires_at > NOW())
		FROM private_rooms WHERE id = ?`, roomID)
	return err == nil && active
}

func (s *Store) ActivateRoomAI(ctx context.Context, userID uint64, roomID string, cost int, until time.Time) (models.User, models.PrivateRoom, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return models.User{}, models.PrivateRoom{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE users SET credits = credits - ? WHERE id = ? AND credits >= ?`, cost, userID, cost)
	if err != nil {
		return models.User{}, models.PrivateRoom{}, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return models.User{}, models.PrivateRoom{}, errors.New("insufficient credits")
	}
	_, err = tx.ExecContext(ctx, `UPDATE private_rooms SET is_ai_active = true, ai_expires_at = ? WHERE id = ?`, until, roomID)
	if err != nil {
		return models.User{}, models.PrivateRoom{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.User{}, models.PrivateRoom{}, err
	}
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return models.User{}, models.PrivateRoom{}, err
	}
	var room models.PrivateRoom
	err = s.db.GetContext(ctx, &room, `SELECT * FROM private_rooms WHERE id = ?`, roomID)
	return user, room, err
}

func (s *Store) AddCreditsByFirebaseUID(ctx context.Context, firebaseUID string, credits int) (models.User, error) {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET credits = credits + ? WHERE firebase_uid = ?`, credits, firebaseUID)
	if err != nil {
		return models.User{}, err
	}
	var user models.User
	err = s.db.GetContext(ctx, &user, `SELECT * FROM users WHERE firebase_uid = ? LIMIT 1`, firebaseUID)
	return user, err
}

func (s *Store) SetVIPByFirebaseUID(ctx context.Context, firebaseUID string, isVIP bool) (models.User, error) {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET is_vip = ? WHERE firebase_uid = ?`, isVIP, firebaseUID)
	if err != nil {
		return models.User{}, err
	}
	var user models.User
	err = s.db.GetContext(ctx, &user, `SELECT * FROM users WHERE firebase_uid = ? LIMIT 1`, firebaseUID)
	return user, err
}

func (s *Store) SearchUsers(ctx context.Context, query string, selfID uint64) ([]models.User, error) {
	var users []models.User
	q := "%" + strings.TrimSpace(query) + "%"
	err := s.db.SelectContext(ctx, &users, `
		SELECT * FROM users
		WHERE id <> ? AND (username LIKE ? OR full_name LIKE ?)
		ORDER BY created_at DESC LIMIT 20`, selfID, q, q)
	return users, err
}

func (s *Store) RequestFriend(ctx context.Context, userID, friendID uint64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO friends (user_id, friend_id, status)
		VALUES (?, ?, 'pending')
		ON DUPLICATE KEY UPDATE status = VALUES(status)`, userID, friendID)
	return err
}
