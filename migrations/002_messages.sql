CREATE TABLE messages (

    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,

    room_id BIGINT UNSIGNED NOT NULL,

    sender_id BIGINT UNSIGNED NOT NULL,

    content TEXT NOT NULL,

    media_id BIGINT UNSIGNED NULL,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_room_created (room_id, created_at),

    INDEX idx_sender (sender_id),

    INDEX idx_media_id (media_id),

    FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE,
    
    FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE SET NULL
);