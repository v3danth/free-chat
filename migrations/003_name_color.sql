-- Members and guests pick the colour their name is shown in.
ALTER TABLE users ADD COLUMN name_color VARCHAR(16) NOT NULL DEFAULT 'stone' AFTER tags;
