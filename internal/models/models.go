package models

import "time"

type User struct {
	ID           uint64    `db:"id" json:"id"`
	PublicID     string    `db:"public_id" json:"public_id"`
	FirebaseUID  *string   `db:"firebase_uid" json:"firebase_uid,omitempty"`
	Email        *string   `db:"email" json:"email,omitempty"`
	PasswordHash *string   `db:"password_hash" json:"-"`
	Username     string    `db:"username" json:"username"`
	FullName     string    `db:"full_name" json:"full_name"`
	AvatarURL    *string   `db:"avatar_url" json:"avatar_url,omitempty"`
	Age          *int      `db:"age" json:"age,omitempty"`
	JobTitle     *string   `db:"job_title" json:"job_title,omitempty"`
	Gender       *string   `db:"gender" json:"gender,omitempty"`
	Partner      *string   `db:"partner" json:"partner,omitempty"`
	Credits      int       `db:"credits" json:"credits"`
	IsVIP        bool      `db:"is_vip" json:"is_vip"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type Thread struct {
	ID             string    `db:"id" json:"id"`
	AuthorID       *uint64   `db:"author_id" json:"author_id,omitempty"`
	Content        string    `db:"content" json:"content"`
	CommentCount   int       `db:"comment_count" json:"comment_count"`
	IsActive       bool      `db:"is_active" json:"is_active"`
	IsPremiumOnly  bool      `db:"is_premium_only" json:"is_premium_only"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	LastActivityAt time.Time `db:"last_activity_at" json:"last_activity_at"`
}

type Comment struct {
	ID             string    `db:"id" json:"id"`
	ThreadID       *string   `db:"thread_id" json:"thread_id,omitempty"`
	RoomID         *string   `db:"room_id" json:"room_id,omitempty"`
	ConversationID *string   `db:"conversation_id" json:"conversation_id,omitempty"`
	UserID         *uint64   `db:"user_id" json:"user_id,omitempty"`
	IsAI           bool      `db:"is_ai" json:"is_ai"`
	AIPersonaType  *string   `db:"ai_persona_type" json:"ai_persona_type,omitempty"`
	Content        string    `db:"content" json:"content"`
	MessageType    string    `db:"message_type" json:"message_type"`
	MediaURL       *string   `db:"media_url" json:"media_url,omitempty"`
	ThumbnailURL   *string   `db:"thumbnail_url" json:"thumbnail_url,omitempty"`
	MimeType       *string   `db:"mime_type" json:"mime_type,omitempty"`
	FileName       *string   `db:"file_name" json:"file_name,omitempty"`
	FileSize       *int64    `db:"file_size" json:"file_size,omitempty"`
	DurationMS     *int      `db:"duration_ms" json:"duration_ms,omitempty"`
	ReplyToID      *string   `db:"reply_to_id" json:"reply_to_id,omitempty"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

type MessageInput struct {
	Content      string  `json:"content"`
	MessageType  string  `json:"message_type"`
	MediaURL     *string `json:"media_url"`
	ThumbnailURL *string `json:"thumbnail_url"`
	MimeType     *string `json:"mime_type"`
	FileName     *string `json:"file_name"`
	FileSize     *int64  `json:"file_size"`
	DurationMS   *int    `json:"duration_ms"`
	ReplyToID    *string `json:"reply_to_id"`
}

type Conversation struct {
	ID            string     `db:"id" json:"id"`
	Type          string     `db:"conversation_type" json:"conversation_type"`
	LastMessageAt *time.Time `db:"last_message_at" json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
}

type PrivateRoom struct {
	ID          string     `db:"id" json:"id"`
	OwnerID     uint64     `db:"owner_id" json:"owner_id"`
	Title       string     `db:"title" json:"title"`
	InviteCode  string     `db:"invite_code" json:"invite_code"`
	IsAIActive  bool       `db:"is_ai_active" json:"is_ai_active"`
	AIExpiresAt *time.Time `db:"ai_expires_at" json:"ai_expires_at,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}

type Friend struct {
	ID        uint64    `db:"id" json:"id"`
	UserID    uint64    `db:"user_id" json:"user_id"`
	FriendID  uint64    `db:"friend_id" json:"friend_id"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
