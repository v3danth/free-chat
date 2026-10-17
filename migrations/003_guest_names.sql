-- Guest names change hands: claiming a name revokes older guests who held
-- it, and a returning guest is told whether that happened. Both look guests
-- up by name.
ALTER TABLE users ADD KEY idx_guest_name (kind, name);
