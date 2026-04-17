USE dedikodu_kazani;

ALTER TABLE comments
  ADD COLUMN IF NOT EXISTS conversation_id CHAR(36) NULL AFTER room_id,
  ADD COLUMN IF NOT EXISTS message_type ENUM('text','emoji','image','video','audio','file') NOT NULL DEFAULT 'text' AFTER content,
  ADD COLUMN IF NOT EXISTS media_url TEXT NULL AFTER message_type,
  ADD COLUMN IF NOT EXISTS thumbnail_url TEXT NULL AFTER media_url,
  ADD COLUMN IF NOT EXISTS mime_type VARCHAR(120) NULL AFTER thumbnail_url,
  ADD COLUMN IF NOT EXISTS file_name VARCHAR(255) NULL AFTER mime_type,
  ADD COLUMN IF NOT EXISTS file_size BIGINT NULL AFTER file_name,
  ADD COLUMN IF NOT EXISTS duration_ms INT NULL AFTER file_size,
  ADD COLUMN IF NOT EXISTS reply_to_id CHAR(36) NULL AFTER duration_ms;

CREATE INDEX IF NOT EXISTS idx_comments_conversation_time ON comments (conversation_id, created_at);

CREATE TABLE IF NOT EXISTS conversations (
  id CHAR(36) NOT NULL,
  conversation_type ENUM('direct') NOT NULL DEFAULT 'direct',
  last_message_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_conversations_last_message (last_message_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS conversation_members (
  conversation_id CHAR(36) NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (conversation_id, user_id),
  KEY idx_conversation_members_user (user_id),
  CONSTRAINT fk_conversation_members_conversation FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
  CONSTRAINT fk_conversation_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS uploads (
  id CHAR(36) NOT NULL,
  owner_id BIGINT UNSIGNED NOT NULL,
  url TEXT NOT NULL,
  file_name VARCHAR(255) NOT NULL,
  mime_type VARCHAR(120) NOT NULL,
  file_size BIGINT NOT NULL DEFAULT 0,
  media_type ENUM('image','video','audio','file') NOT NULL DEFAULT 'file',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_uploads_owner (owner_id, created_at),
  CONSTRAINT fk_uploads_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
