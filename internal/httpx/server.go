package httpx

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"dedikodu-kazani/backend/internal/ai"
	"dedikodu-kazani/backend/internal/auth"
	"dedikodu-kazani/backend/internal/config"
	"dedikodu-kazani/backend/internal/models"
	"dedikodu-kazani/backend/internal/payments"
	"dedikodu-kazani/backend/internal/realtime"
	"dedikodu-kazani/backend/internal/repository"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	cfg      config.Config
	store    *repository.Store
	firebase auth.FirebaseVerifier
	ai       ai.Provider
	hub      *realtime.Hub
}

func New(cfg config.Config, store *repository.Store, firebase auth.FirebaseVerifier, aiProvider ai.Provider, hub *realtime.Hub) *gin.Engine {
	s := &Server{cfg: cfg, store: store, firebase: firebase, ai: aiProvider, hub: hub}
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true, "app": cfg.AppName}) })
	r.Static("/uploads", "./storage/uploads")
	api := r.Group("/api")
	api.POST("/auth/login", s.loginFirebase)
	api.POST("/auth/register", s.registerEmail)
	api.POST("/auth/email-login", s.loginEmail)
	api.POST("/webhooks/revenuecat", s.revenueCat)

	protected := api.Group("")
	protected.Use(s.requireAuth())
	protected.GET("/user/me", s.me)
	protected.PUT("/user/me", s.updateProfile)
	protected.GET("/users/search", s.searchUsers)
	protected.POST("/friends/:id", s.requestFriend)
	protected.GET("/threads", s.listThreads)
	protected.POST("/threads", s.createThread)
	protected.GET("/threads/:id/comments", s.listComments)
	protected.POST("/threads/:id/comments", s.createComment)
	protected.POST("/rooms", s.createRoom)
	protected.POST("/rooms/join", s.joinRoom)
	protected.GET("/rooms/:id/comments", s.listRoomComments)
	protected.POST("/rooms/:id/comments", s.createRoomComment)
	protected.POST("/rooms/:id/invite-bot", s.inviteBot)
	protected.POST("/uploads", s.uploadMedia)
	protected.POST("/dms/:user_id", s.openDirectMessage)
	protected.GET("/conversations/:id/messages", s.listConversationMessages)
	protected.POST("/conversations/:id/messages", s.createConversationMessage)
	protected.GET("/ws", s.websocket)
	return r
}

func (s *Server) loginFirebase(c *gin.Context) {
	var req struct {
		FirebaseToken string `json:"firebase_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if s.firebase == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "firebase henuz yapilandirilmadi"})
		return
	}
	identity, err := s.firebase.VerifyIDToken(c.Request.Context(), req.FirebaseToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "firebase token gecersiz"})
		return
	}
	user, err := s.store.UpsertFirebaseUser(c.Request.Context(), identity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	token, _ := auth.Issue(s.cfg.JWTSecret, user.ID)
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func (s *Server) registerEmail(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		FullName string `json:"full_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	user, err := s.store.CreateEmailUser(c.Request.Context(), req.Email, string(hash), strings.TrimSpace(req.FullName))
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "bu e-posta zaten kullaniliyor olabilir"})
		return
	}
	token, _ := auth.Issue(s.cfg.JWTSecret, user.ID)
	c.JSON(http.StatusCreated, gin.H{"token": token, "user": user})
}

func (s *Server) loginEmail(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, hash, err := s.store.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email veya sifre hatali"})
		return
	}
	token, _ := auth.Issue(s.cfg.JWTSecret, user.ID)
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func (s *Server) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" && c.Query("token") != "" {
			header = "Bearer " + c.Query("token")
		}
		if header == "" {
			header = websocketProtocolToken(c.GetHeader("Sec-WebSocket-Protocol"))
		}
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token gerekli"})
			return
		}
		claims, err := auth.Parse(s.cfg.JWTSecret, strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token gecersiz"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

func websocketProtocolToken(value string) string {
	parts := strings.Split(value, ",")
	for i, part := range parts {
		if strings.TrimSpace(part) == "Bearer" && i+1 < len(parts) {
			return "Bearer " + strings.TrimSpace(parts[i+1])
		}
	}
	return ""
}

func userID(c *gin.Context) uint64 {
	value, _ := c.Get("user_id")
	id, _ := value.(uint64)
	return id
}

func (s *Server) me(c *gin.Context) {
	user, err := s.store.GetUserByID(c.Request.Context(), userID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "kullanici bulunamadi"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (s *Server) updateProfile(c *gin.Context) {
	var req struct {
		FullName     string  `json:"full_name" binding:"required"`
		Age          *int    `json:"age"`
		JobTitle     *string `json:"job_title"`
		Gender       *string `json:"gender"`
		Partner      *string `json:"partner"`
		AvatarURL    *string `json:"avatar_url"`
		Bio          *string `json:"bio"`
		LanguageCode *string `json:"language_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := s.store.UpdateProfile(c.Request.Context(), userID(c), req.FullName, req.Age, req.JobTitle, req.Gender, req.Partner, req.AvatarURL, req.Bio, req.LanguageCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (s *Server) listThreads(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	threads, err := s.store.ListThreads(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": threads})
}

func (s *Server) createThread(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	thread, err := s.store.CreateThread(c.Request.Context(), userID(c), req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, thread)
}

func (s *Server) listComments(c *gin.Context) {
	comments, err := s.store.ListComments(c.Request.Context(), c.Param("id"), 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": comments})
}

func (s *Server) createComment(c *gin.Context) {
	var req models.MessageInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Content) == "" && req.MediaURL == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mesaj veya medya gerekli"})
		return
	}
	uid := userID(c)
	comment, err := s.store.CreateThreadComment(c.Request.Context(), c.Param("id"), &uid, req, false, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	room := "thread:" + c.Param("id")
	s.hub.Publish(c.Request.Context(), room, realtime.Event{Event: "MESSAGE_RECEIVED", Payload: comment})
	if strings.TrimSpace(req.Content) != "" {
		go s.respondWithAI(context.Background(), c.Param("id"), req.Content)
	}
	c.JSON(http.StatusCreated, comment)
}

func (s *Server) respondWithAI(ctx context.Context, threadID, latest string) {
	room := "thread:" + threadID
	persona := "SukriyeTeyze"
	s.hub.Publish(ctx, room, realtime.Event{Event: "AI_TYPING", Payload: gin.H{"persona": persona, "typing": true}})
	time.Sleep(3 * time.Second)
	reply, err := s.ai.GenerateReply(ctx, persona, nil, latest)
	if err != nil || reply == "" {
		return
	}
	delay := time.Duration(len([]rune(reply))/12) * time.Second
	if delay < 2*time.Second {
		delay = 2 * time.Second
	}
	if delay > 7*time.Second {
		delay = 7 * time.Second
	}
	time.Sleep(delay)
	comment, err := s.store.CreateThreadComment(ctx, threadID, nil, models.MessageInput{Content: reply, MessageType: "text"}, true, &persona)
	if err == nil {
		s.hub.Publish(ctx, room, realtime.Event{Event: "MESSAGE_RECEIVED", Payload: comment})
	}
}

func (s *Server) createRoom(c *gin.Context) {
	var req struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	room, err := s.store.CreateRoom(c.Request.Context(), userID(c), req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, room)
}

func (s *Server) joinRoom(c *gin.Context) {
	var req struct {
		InviteCode string `json:"invite_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	room, err := s.store.JoinRoom(c.Request.Context(), userID(c), req.InviteCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "oda bulunamadi"})
		return
	}
	c.JSON(http.StatusOK, room)
}

func (s *Server) listRoomComments(c *gin.Context) {
	comments, err := s.store.ListRoomComments(c.Request.Context(), c.Param("id"), 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": comments})
}

func (s *Server) createRoomComment(c *gin.Context) {
	var req models.MessageInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Content) == "" && req.MediaURL == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mesaj veya medya gerekli"})
		return
	}
	uid := userID(c)
	comment, err := s.store.CreateRoomComment(c.Request.Context(), c.Param("id"), &uid, req, false, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	room := "room:" + c.Param("id")
	s.hub.Publish(c.Request.Context(), room, realtime.Event{Event: "MESSAGE_RECEIVED", Payload: comment})
	if strings.TrimSpace(req.Content) != "" && s.store.IsRoomAIActive(c.Request.Context(), c.Param("id")) {
		go s.respondWithRoomAI(context.Background(), c.Param("id"), req.Content)
	}
	c.JSON(http.StatusCreated, comment)
}

func (s *Server) respondWithRoomAI(ctx context.Context, roomID, latest string) {
	room := "room:" + roomID
	persona := "Gazlayici"
	s.hub.Publish(ctx, room, realtime.Event{Event: "AI_TYPING", Payload: gin.H{"persona": persona, "typing": true}})
	reply, err := s.ai.GenerateReply(ctx, persona, nil, latest)
	if err != nil || reply == "" {
		return
	}
	delay := time.Duration(len([]rune(reply))/12) * time.Second
	if delay < 3*time.Second {
		delay = 3 * time.Second
	}
	if delay > 7*time.Second {
		delay = 7 * time.Second
	}
	time.Sleep(delay)
	comment, err := s.store.CreateRoomComment(ctx, roomID, nil, models.MessageInput{Content: reply, MessageType: "text"}, true, &persona)
	if err == nil {
		s.hub.Publish(ctx, room, realtime.Event{Event: "MESSAGE_RECEIVED", Payload: comment})
	}
}

func (s *Server) uploadMedia(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dosya gerekli"})
		return
	}
	mediaType := c.DefaultPostForm("media_type", "file")
	if mediaType == "" {
		mediaType = "file"
	}
	id := fmt.Sprintf("%d-%s", time.Now().UnixNano(), filepath.Base(file.Filename))
	dir := filepath.Join("storage", "uploads", fmt.Sprintf("%d", userID(c)))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	dst := filepath.Join(dir, id)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	url := "/" + filepath.ToSlash(dst)
	mimeType := file.Header.Get("Content-Type")
	if err := s.store.SaveUpload(c.Request.Context(), userID(c), url, file.Filename, mimeType, mediaType, file.Size); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"url": url, "file_name": file.Filename, "mime_type": mimeType,
		"file_size": file.Size, "media_type": mediaType,
	})
}

func (s *Server) openDirectMessage(c *gin.Context) {
	otherID, _ := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if otherID == 0 || otherID == userID(c) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gecersiz kullanici"})
		return
	}
	conversation, err := s.store.GetOrCreateDirectConversation(c.Request.Context(), userID(c), otherID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, conversation)
}

func (s *Server) listConversationMessages(c *gin.Context) {
	messages, err := s.store.ListConversationMessages(c.Request.Context(), c.Param("id"), userID(c), 100)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": messages})
}

func (s *Server) createConversationMessage(c *gin.Context) {
	var req models.MessageInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Content) == "" && req.MediaURL == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mesaj veya medya gerekli"})
		return
	}
	message, err := s.store.CreateConversationMessage(c.Request.Context(), c.Param("id"), userID(c), req)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	room := "conversation:" + c.Param("id")
	s.hub.Publish(c.Request.Context(), room, realtime.Event{Event: "MESSAGE_RECEIVED", Payload: message})
	c.JSON(http.StatusCreated, message)
}

func (s *Server) inviteBot(c *gin.Context) {
	until := time.Now().Add(s.cfg.AIRoomDuration)
	user, room, err := s.store.ActivateRoomAI(c.Request.Context(), userID(c), c.Param("id"), s.cfg.AIRoomCost, until)
	if err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "jeton yetersiz"})
		return
	}
	s.hub.PushUser(user.ID, realtime.Event{Event: "BALANCE_UPDATED", Payload: gin.H{"credits": user.Credits, "is_vip": user.IsVIP}})
	c.JSON(http.StatusOK, room)
}

func (s *Server) searchUsers(c *gin.Context) {
	users, err := s.store.SearchUsers(c.Request.Context(), c.Query("q"), userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": users})
}

func (s *Server) requestFriend(c *gin.Context) {
	friendID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if friendID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gecersiz kullanici"})
		return
	}
	if err := s.store.RequestFriend(c.Request.Context(), userID(c), friendID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "pending"})
}

func (s *Server) revenueCat(c *gin.Context) {
	if s.cfg.RevenueCatWebhookSecret != "" {
		expected := "Bearer " + s.cfg.RevenueCatWebhookSecret
		if subtle.ConstantTimeCompare([]byte(c.GetHeader("Authorization")), []byte(expected)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
	}
	var payload payments.RevenueCatWebhook
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	credits := payments.CreditsForProduct(payload.Event.ProductID)
	if credits > 0 {
		user, err := s.store.AddCreditsByFirebaseUID(c.Request.Context(), payload.Event.AppUserID, credits)
		if err == nil {
			s.hub.PushUser(user.ID, realtime.Event{Event: "BALANCE_UPDATED", Payload: gin.H{"credits": user.Credits, "is_vip": user.IsVIP}})
		}
	}
	if payments.IsVIPEvent(payload.Event.Type) || payments.IsVIPCancelEvent(payload.Event.Type) {
		isVIP := payments.IsVIPEvent(payload.Event.Type)
		user, err := s.store.SetVIPByFirebaseUID(c.Request.Context(), payload.Event.AppUserID, isVIP)
		if err == nil {
			s.hub.PushUser(user.ID, realtime.Event{Event: "BALANCE_UPDATED", Payload: gin.H{"credits": user.Credits, "is_vip": user.IsVIP}})
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) websocket(c *gin.Context) {
	rooms := strings.Split(c.DefaultQuery("rooms", "global"), ",")
	s.hub.Serve(c.Writer, c.Request, userID(c), rooms)
}
