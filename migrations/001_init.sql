CREATE DATABASE IF NOT EXISTS dedikodu_kazani
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE dedikodu_kazani;

CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  public_id CHAR(36) NOT NULL,
  firebase_uid VARCHAR(191) NULL,
  email VARCHAR(191) NULL,
  password_hash VARCHAR(255) NULL,
  username VARCHAR(64) NOT NULL,
  full_name VARCHAR(120) NOT NULL,
  avatar_url TEXT NULL,
  bio TEXT NULL,
  language_code VARCHAR(12) NULL DEFAULT 'tr',
  age INT NULL,
  job_title VARCHAR(120) NULL,
  gender VARCHAR(40) NULL,
  partner VARCHAR(120) NULL,
  credits INT NOT NULL DEFAULT 0,
  is_vip TINYINT(1) NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_users_public_id (public_id),
  UNIQUE KEY uq_users_firebase_uid (firebase_uid),
  UNIQUE KEY uq_users_email (email),
  UNIQUE KEY uq_users_username (username),
  KEY idx_users_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS threads (
  id CHAR(36) NOT NULL,
  author_id BIGINT UNSIGNED NULL,
  content TEXT NOT NULL,
  comment_count INT NOT NULL DEFAULT 0,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  is_premium_only TINYINT(1) NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_activity_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_threads_author (author_id),
  KEY idx_threads_activity (is_active, last_activity_at),
  CONSTRAINT fk_threads_author FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS comments (
  id CHAR(36) NOT NULL,
  thread_id CHAR(36) NULL,
  room_id CHAR(36) NULL,
  user_id BIGINT UNSIGNED NULL,
  is_ai TINYINT(1) NOT NULL DEFAULT 0,
  ai_persona_type VARCHAR(64) NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_comments_thread_time (thread_id, created_at),
  KEY idx_comments_room_time (room_id, created_at),
  KEY idx_comments_user (user_id),
  CONSTRAINT fk_comments_thread FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE,
  CONSTRAINT fk_comments_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS private_rooms (
  id CHAR(36) NOT NULL,
  owner_id BIGINT UNSIGNED NOT NULL,
  title VARCHAR(120) NOT NULL,
  invite_code VARCHAR(24) NOT NULL,
  is_ai_active TINYINT(1) NOT NULL DEFAULT 0,
  ai_expires_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_private_rooms_invite_code (invite_code),
  KEY idx_private_rooms_owner (owner_id),
  CONSTRAINT fk_private_rooms_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS room_members (
  room_id CHAR(36) NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  role ENUM('owner', 'member') NOT NULL DEFAULT 'member',
  joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (room_id, user_id),
  KEY idx_room_members_user (user_id),
  CONSTRAINT fk_room_members_room FOREIGN KEY (room_id) REFERENCES private_rooms(id) ON DELETE CASCADE,
  CONSTRAINT fk_room_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS friends (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  friend_id BIGINT UNSIGNED NOT NULL,
  status ENUM('pending', 'accepted', 'blocked') NOT NULL DEFAULT 'pending',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_friends_pair (user_id, friend_id),
  KEY idx_friends_friend_status (friend_id, status),
  CONSTRAINT fk_friends_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_friends_friend FOREIGN KEY (friend_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS prompt_templates (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  persona_type VARCHAR(64) NOT NULL,
  system_prompt TEXT NOT NULL,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_prompt_persona (persona_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO prompt_templates (persona_type, system_prompt) VALUES
('SukriyeTeyze', '50 yasinda ahlak bekcisi Sukriye Teyze gibi kisa, komik ve igneleyici cevap ver.'),
('Gazlayici', 'Tartismayi eglenceli sekilde gazlayan ama kimseyi hedef gostermeyen cevaplar ver.'),
('Mantikci', 'Olaylara sakin ve mantikli yaklas, gerekirse espriyle tansiyonu dusur.');
