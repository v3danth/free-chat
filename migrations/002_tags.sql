-- Free-form "here to" tags replace the fixed intent. Existing users keep
-- what they had picked, as their first tag.
ALTER TABLE users ADD COLUMN tags JSON NOT NULL DEFAULT (JSON_ARRAY()) AFTER age;
UPDATE users SET tags = JSON_ARRAY(REPLACE(intent, '_', ' '));
ALTER TABLE users DROP COLUMN intent;
