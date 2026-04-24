DROP INDEX IF EXISTS idx_group_invitations_invitee_status;
DROP INDEX IF EXISTS idx_group_invitations_group_status;
DROP TABLE IF EXISTS group_invitations;

DROP INDEX IF EXISTS idx_group_join_requests_user_status;
DROP INDEX IF EXISTS idx_group_join_requests_group_status;
DROP TABLE IF EXISTS group_join_requests;

DROP INDEX IF EXISTS idx_groups_creator_id;

ALTER TABLE group_members RENAME TO group_members_new;
CREATE TABLE group_members (
    group_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    PRIMARY KEY (group_id, user_id),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
INSERT INTO group_members (group_id, user_id)
SELECT group_id, user_id
FROM group_members_new;
DROP TABLE group_members_new;

ALTER TABLE groups RENAME TO groups_new;
CREATE TABLE groups (
    id TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO groups (id, created_at)
SELECT id, created_at
FROM groups_new;
DROP TABLE groups_new;
