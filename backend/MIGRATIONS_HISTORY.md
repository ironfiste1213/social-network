# Database Migration History (SQLite)

This document describes the migrations applied since the start of the project.

Migration location:
- `backend/pkg/db/migrations/sqlite`

Execution order:
- migrations are run in filename order (from `000001` to `000012`) in `backend/server.go` via `sqlite.RunMigrations(...)`.

## Timeline

### 000001_create_users_table

Purpose:
- Introduce the core `users` table.

Main changes:
- Creates `users` with identity and profile fields:
  - `id`, `email` (unique), `password_hash`
  - `first_name`, `last_name`, `date_of_birth`
  - optional profile fields (`avatar_path`, `nickname`, `about_me`)
  - `profile_visibility` with allowed values `public|private`
  - timestamps (`created_at`, `updated_at`)

Down migration:
- Drops `users`.

### 000002_create_sessions_table

Purpose:
- Add server-side session storage for authentication.

Main changes:
- Creates `sessions` table with:
  - `id` (session id)
  - `user_id` (FK to `users.id`, cascade delete)
  - `expires_at`, `created_at`
- Adds index `idx_sessions_user_id`.

Down migration:
- Drops the index and `sessions`.

### 000003_create_follow_requests_table

Purpose:
- Support private-account follow requests.

Main changes:
- Creates `follow_requests` table:
  - `sender_id`, `receiver_id` (both FK to `users`)
  - `status` (`pending|accepted|declined`)
  - `created_at`, `responded_at`
- Adds unique constraint `(sender_id, receiver_id)`.
- Adds index `idx_follow_requests_receiver_id`.

Down migration:
- Drops index and table.

### 000004_create_followers_table

Purpose:
- Store accepted follower relations.

Main changes:
- Creates `followers` join table:
  - `follower_id`, `following_id` (composite PK)
  - both FK to `users` with cascade delete
  - `created_at`
- Adds index `idx_followers_following_id`.

Down migration:
- Drops index and table.

### 000005_create_posts_table

Purpose:
- Add social posts.

Main changes:
- Creates `posts` table:
  - `id`, `author_id` (FK users)
  - `body`, optional `image_path`
  - `privacy` (`public|followers|selected_followers`)
  - `created_at`, `updated_at`
- Adds index `idx_posts_author_id`.

Down migration:
- Drops index and table.

### 000006_create_comments_table

Purpose:
- Add comments on posts.

Main changes:
- Creates `comments` table:
  - `id`, `post_id` (FK posts), `author_id` (FK users)
  - `body`, optional `image_path`
  - `created_at`, `updated_at`
- Adds index `idx_comments_post_id`.

Down migration:
- Drops index and table.

### 000007_create_users_nickname_index

Purpose:
- Improve nickname lookups/search performance.

Main changes:
- Adds case-insensitive index on nickname:
  - `idx_users_nickname_nocase` on `users(nickname COLLATE NOCASE)`.

Down migration:
- Drops the index.

### 000008_create_post_viewers_table

Purpose:
- Support selected-viewers privacy for posts.

Main changes:
- Creates `post_viewers` join table:
  - `post_id`, `user_id` (composite PK)
  - FK to `posts` and `users` with cascade delete

Down migration:
- Drops `post_viewers`.

### 000009_make_users_nickname_unique_nocase

Purpose:
- Enforce nickname uniqueness (case-insensitive) while still allowing empty/null nicknames.

Main changes:
- Drops old non-unique index `idx_users_nickname_nocase`.
- Creates partial unique index:
  - `idx_users_nickname_unique_nocase`
  - unique on `nickname COLLATE NOCASE`
  - applies only when nickname is not null and not blank (`TRIM(nickname) != ''`).

Down migration:
- Reverts to non-unique case-insensitive nickname index.

### 000010_add_group_id_to_posts

Purpose:
- Prepare posts to optionally belong to groups.

Main changes:
- Adds nullable `group_id` column to `posts` referencing `groups(id)`.
- Adds index `idx_posts_group_id`.

Important note:
- This migration appears before groups tables are created. SQLite allows this schema declaration, and the group tables arrive in later migrations.

Down migration:
- Rebuilds `posts` table without `group_id` (SQLite cannot simply drop a column).
- Steps: create backup table -> copy data -> drop old table -> rename backup -> recreate index.

### 000011_create_groups_tables

Purpose:
- Introduce base group entities.

Main changes:
- Creates `groups` with basic fields:
  - `id`, `created_at`
- Creates `group_members` join table:
  - `group_id`, `user_id` (composite PK)
  - FK to `groups` and `users`
- Adds index `idx_group_members_user_id`.

Down migration:
- Drops index, `group_members`, and `groups`.

### 000012_finalize_groups_schema

Purpose:
- Complete groups feature with metadata, roles, join requests, and invitations.

Main changes:
- Extends `groups`:
  - adds `title`, `group_description`, `creator_id` (FK users)
  - adds index `idx_groups_creator_id`
- Extends `group_members`:
  - adds `member_role` (`creator|member`)
  - adds `joined_at`
- Creates `group_join_requests` table + status indexes.
- Creates `group_invitations` table + status indexes.

Down migration:
- Drops invitation/request tables and indexes.
- Rebuilds `group_members` and `groups` to their old minimal schema (without new columns), preserving base data.

## Evolution by Feature

1. Identity/auth foundation:
- `000001` users
- `000002` sessions

2. Social graph:
- `000003` follow requests
- `000004` followers

3. Content system:
- `000005` posts
- `000006` comments
- `000008` post viewers for private audience selection

4. Search/consistency improvements:
- `000007` nickname index
- `000009` unique case-insensitive nickname rules

5. Group feature rollout:
- `000010` posts prepared for `group_id`
- `000011` base groups/group_members
- `000012` full groups schema (creator metadata, roles, requests, invitations)

