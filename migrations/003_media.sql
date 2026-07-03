CREATE TABLE media (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,

    user_id BIGINT UNSIGNED NOT NULL,

    media_type ENUM('image', 'gif', 'voice') NOT NULL,

    file_name VARCHAR(255) NOT NULL,

    file_path VARCHAR(512) NOT NULL,

    file_size BIGINT UNSIGNED NOT NULL,

    mime_type VARCHAR(100) NOT NULL,

    width INT UNSIGNED NULL,

    height INT UNSIGNED NULL,

    duration_ms INT UNSIGNED NULL,

    flag_count INT UNSIGNED NOT NULL DEFAULT 0,

    is_flagged BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_user (user_id),
    INDEX idx_flagged (is_flagged, flag_count),

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE media_flags (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,

    media_id BIGINT UNSIGNED NOT NULL,

    reporter_id BIGINT UNSIGNED NOT NULL,

    reason ENUM(
        'inappropriate',
        'spam',
        -- 'nudity',
        'violence',
        -- 'copyright',
        'other'
    ) NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uniq_media_reporter (media_id, reporter_id),

    INDEX idx_media (media_id),

    FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE CASCADE,
    FOREIGN KEY (reporter_id) REFERENCES users(id) ON DELETE CASCADE
);

ALTER TABLE messages 
    ADD COLUMN media_id BIGINT UNSIGNED NULL AFTER content,
    ADD INDEX idx_media_id (media_id),
    ADD CONSTRAINT fk_message_media 
        FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE SET NULL;