-- Warmth, yearning and safety (docs/warmth-yearning-safety.md).
--
-- Settings people choose for their card: a glow around their face, who may
-- knock, and whether strangers see their gender.
ALTER TABLE users
    ADD COLUMN aura        VARCHAR(16) NULL AFTER about,
    ADD COLUMN knocks_from ENUM('anyone', 'members', 'nobody') NOT NULL DEFAULT 'anyone' AFTER aura,
    ADD COLUMN hide_gender BOOLEAN NOT NULL DEFAULT FALSE AFTER knocks_from;

-- Letters are slow mail between members who have talked: written in an open
-- chat, delivered hours later. Ids come from internal/id, so retention
-- deletes by primary-key range like messages.
CREATE TABLE letters (
    id           BIGINT UNSIGNED NOT NULL PRIMARY KEY,
    sender_id    BIGINT UNSIGNED NOT NULL,
    recipient_id BIGINT UNSIGNED NOT NULL,
    sender_name  VARCHAR(32) NOT NULL,
    body         VARCHAR(1000) NOT NULL,
    deliver_at   DATETIME NOT NULL,
    delivered_at DATETIME NULL,

    KEY idx_due (delivered_at, deliver_at),
    KEY idx_pending (sender_id, recipient_id, delivered_at),
    CONSTRAINT fk_letters_sender FOREIGN KEY (sender_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_letters_recipient FOREIGN KEY (recipient_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;
