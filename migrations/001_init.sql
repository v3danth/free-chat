CREATE TABLE users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,

    user_type ENUM('guest', 'registered') NOT NULL,

    username VARCHAR(32) NOT NULL UNIQUE,

    gender ENUM(
        'male',
        'female',
        'non-binary',
        'femboy',
        'other',
        'couple'
    ) NOT NULL,

    age TINYINT UNSIGNED NOT NULL,

    about VARCHAR(50) NOT NULL DEFAULT '',

    email VARCHAR(255) NULL UNIQUE,

    password_hash VARCHAR(255) NULL,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_user_type (user_type),
    INDEX idx_email (email),
    INDEX idx_username (username)
);