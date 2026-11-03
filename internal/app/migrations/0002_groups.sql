CREATE TABLE relation_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL CHECK (type IN ('形音相似', '近义', '反义', '词族', '屈折变化', '拼写变体', '语义相关')),
    member_key TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    UNIQUE (type, member_key)
);

CREATE TABLE group_members (
    group_id INTEGER NOT NULL REFERENCES relation_groups(id),
    entry_id INTEGER NOT NULL REFERENCES entries(id),
    PRIMARY KEY (group_id, entry_id)
);

CREATE INDEX group_members_entry ON group_members(entry_id, group_id);
