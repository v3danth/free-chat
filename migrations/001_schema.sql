-- Free Online Chat India: schema v2.
-- Fresh install only: `make mysql-reset` drops and recreates the database.
--
-- Design notes
--   * Guests are one-visit: names are not unique for guests, only for members.
--   * Messages are an append-only log with app-generated, time-ordered ids
--     (internal/id), so the server can broadcast before the row is written.
--     Retention deletes by primary-key range.
--   * Every timestamp is UTC (the connection sets time_zone = '+00:00').

SET NAMES utf8mb4;

CREATE TABLE users (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    kind           ENUM('guest', 'member') NOT NULL,
    role           ENUM('user', 'moderator', 'admin') NOT NULL DEFAULT 'user',
    name           VARCHAR(32) NOT NULL,
    gender         ENUM('male', 'female', 'non-binary', 'femboy', 'other', 'couple') NOT NULL,
    age            TINYINT UNSIGNED NOT NULL,
    intent         ENUM('talk', 'flirt', 'night_owl', 'vent', 'something_real') NOT NULL DEFAULT 'talk',
    about          VARCHAR(140) NOT NULL DEFAULT '',
    -- Written by the user ("Mumbai"); optional.
    location       VARCHAR(40) NOT NULL DEFAULT '',
    -- ISO 3166-1 alpha-2 from GeoIP at sign-up; NULL when unknown.
    country_code   CHAR(2) NULL,
    photo_media_id BIGINT UNSIGNED NULL,
    email          VARCHAR(255) NULL,
    password_hash  VARCHAR(255) NULL,
    -- Member names are unique; guest names are not (NULLs never collide).
    member_name    VARCHAR(32) AS (CASE WHEN kind = 'member' THEN name END) STORED,
    -- Bumped to revoke every token the user holds (ban, role change).
    token_version  INT UNSIGNED NOT NULL DEFAULT 0,
    banned_until   DATETIME NULL,
    muted_until    DATETIME NULL,
    -- HMAC of the client IP; used only for IP bans, never stored raw.
    ip_hash        BINARY(32) NULL,
    last_seen_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uniq_email (email),
    UNIQUE KEY uniq_member_name (member_name),
    KEY idx_guest_sweep (kind, last_seen_at),
    CONSTRAINT chk_age CHECK (age BETWEEN 18 AND 99),
    CONSTRAINT chk_member_credentials CHECK (kind = 'guest' OR (email IS NOT NULL AND password_hash IS NOT NULL))
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- Rooms are staged with is_live = FALSE and switched on when ready.
CREATE TABLE rooms (
    id         INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    slug       VARCHAR(32) NOT NULL,
    name       VARCHAR(64) NOT NULL,
    is_live    BOOLEAN NOT NULL DEFAULT FALSE,
    created_by BIGINT UNSIGNED NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uniq_slug (slug),
    CONSTRAINT fk_rooms_creator FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE SET NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

INSERT INTO rooms (id, slug, name, is_live) VALUES (1, 'main', 'Main Room', TRUE);

-- Images only. Files live on disk as <variant>/<file_key>.jpg.
CREATE TABLE media (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    owner_id   BIGINT UNSIGNED NOT NULL,
    file_key   CHAR(26) NOT NULL,
    sha256     BINARY(32) NOT NULL,
    width      SMALLINT UNSIGNED NOT NULL,
    height     SMALLINT UNSIGNED NOT NULL,
    bytes      INT UNSIGNED NOT NULL,
    removed_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uniq_file_key (file_key),
    KEY idx_sha256 (sha256),
    CONSTRAINT fk_media_owner FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB;

ALTER TABLE users
    ADD CONSTRAINT fk_users_photo FOREIGN KEY (photo_media_id) REFERENCES media (id) ON DELETE SET NULL;

-- A removed image can never be uploaded again, by anyone.
CREATE TABLE banned_media_hashes (
    sha256     BINARY(32) NOT NULL PRIMARY KEY,
    created_by BIGINT UNSIGNED NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_banned_hash_creator FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE SET NULL
) ENGINE = InnoDB;

-- Exactly one of room_id / recipient_id is set; enforced by the app because
-- MySQL forbids CHECK constraints on columns with FK referential actions.
CREATE TABLE messages (
    id           BIGINT UNSIGNED NOT NULL PRIMARY KEY,
    room_id      INT UNSIGNED NULL,
    recipient_id BIGINT UNSIGNED NULL,
    sender_id    BIGINT UNSIGNED NOT NULL,
    sender_name  VARCHAR(32) NOT NULL,
    body         VARCHAR(1000) NOT NULL DEFAULT '',
    media_id     BIGINT UNSIGNED NULL,
    hidden_at    DATETIME NULL,

    KEY idx_room (room_id, id),
    KEY idx_sender (sender_id, id),
    CONSTRAINT fk_messages_room FOREIGN KEY (room_id) REFERENCES rooms (id) ON DELETE CASCADE,
    CONSTRAINT fk_messages_recipient FOREIGN KEY (recipient_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_messages_sender FOREIGN KEY (sender_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_messages_media FOREIGN KEY (media_id) REFERENCES media (id) ON DELETE SET NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

CREATE TABLE blocks (
    blocker_id BIGINT UNSIGNED NOT NULL,
    blocked_id BIGINT UNSIGNED NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (blocker_id, blocked_id),
    KEY idx_blocked (blocked_id),
    CONSTRAINT fk_blocks_blocker FOREIGN KEY (blocker_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_blocks_blocked FOREIGN KEY (blocked_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB;

-- evidence is a snapshot taken at report time, so it survives deletion and
-- retention purges of the reported content.
CREATE TABLE reports (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    reporter_id    BIGINT UNSIGNED NULL,
    target_type    ENUM('message', 'media', 'user') NOT NULL,
    target_id      BIGINT UNSIGNED NOT NULL,
    target_user_id BIGINT UNSIGNED NULL,
    reason         ENUM('spam', 'harassment', 'nudity', 'violence', 'hate', 'underage', 'scam', 'other') NOT NULL,
    note           VARCHAR(500) NOT NULL DEFAULT '',
    evidence       JSON NOT NULL,
    status         ENUM('open', 'actioned', 'dismissed') NOT NULL DEFAULT 'open',
    handled_by     BIGINT UNSIGNED NULL,
    handled_at     DATETIME NULL,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uniq_reporter_target (reporter_id, target_type, target_id),
    KEY idx_queue (status, id),
    KEY idx_target (target_type, target_id, status),
    CONSTRAINT fk_reports_reporter FOREIGN KEY (reporter_id) REFERENCES users (id) ON DELETE SET NULL,
    CONSTRAINT fk_reports_target_user FOREIGN KEY (target_user_id) REFERENCES users (id) ON DELETE SET NULL,
    CONSTRAINT fk_reports_handler FOREIGN KEY (handled_by) REFERENCES users (id) ON DELETE SET NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- Append-only audit log of every moderation action.
CREATE TABLE mod_actions (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    actor_id       BIGINT UNSIGNED NULL,
    action         ENUM('remove_message', 'remove_media', 'kick', 'mute', 'ban', 'unban', 'set_role',
                        'add_word', 'remove_word', 'dismiss_report', 'auto_hide') NOT NULL,
    target_user_id BIGINT UNSIGNED NULL,
    target_name    VARCHAR(32) NOT NULL DEFAULT '',
    target_id      BIGINT UNSIGNED NULL,
    detail         VARCHAR(500) NOT NULL DEFAULT '',
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    KEY idx_target_user (target_user_id),
    CONSTRAINT fk_mod_actions_actor FOREIGN KEY (actor_id) REFERENCES users (id) ON DELETE SET NULL,
    CONSTRAINT fk_mod_actions_target FOREIGN KEY (target_user_id) REFERENCES users (id) ON DELETE SET NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- The live word filter. A phrase (several words) can only block.
CREATE TABLE banned_words (
    id         INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    word       VARCHAR(64) NOT NULL,
    action     ENUM('mask', 'block') NOT NULL,
    created_by BIGINT UNSIGNED NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uniq_word (word),
    CONSTRAINT fk_banned_words_creator FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE SET NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- Short-lived by design: Indian mobile carriers put many users behind one
-- IP (CGNAT), so an IP ban always has an expiry.
CREATE TABLE ip_bans (
    ip_hash    BINARY(32) NOT NULL PRIMARY KEY,
    expires_at DATETIME NOT NULL,
    reason     VARCHAR(200) NOT NULL DEFAULT '',
    created_by BIGINT UNSIGNED NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    KEY idx_expires (expires_at),
    CONSTRAINT fk_ip_bans_creator FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE SET NULL
) ENGINE = InnoDB;
