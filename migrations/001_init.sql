CREATE TABLE users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,

    user_type ENUM('guest', 'registered') NOT NULL,

    username VARCHAR(32) NOT NULL,

    gender ENUM(
        'male',
        'female',
        'non-binary',
        'femboy',
        'other',
        'couple' -- can we add more options here?
    ) NOT NULL,

    age TINYINT UNSIGNED NOT NULL,

    about VARCHAR(50) NOT NULL DEFAULT '',

    email VARCHAR(255) NULL UNIQUE,

    password_hash VARCHAR(255) NULL,

    status ENUM('active', 'inactive') NOT NULL DEFAULT 'active',

    last_seen_at TIMESTAMP NULL,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_user_type (user_type),
    INDEX idx_email (email),
    INDEX idx_guest_active (username, user_type, status),
    INDEX idx_status_last_seen (status, last_seen_at)
);