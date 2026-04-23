package users

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"social-network/backend/pkg/db/sqlite"
)

func TestUpdateProfileRejectsDuplicateNicknameCaseInsensitive(t *testing.T) {
	service, repo := newTestService(t)
	ctx := context.Background()

	userOne := insertTestUser(t, repo, ctx, "user-1", "first@example.com", "Nickname")
	_ = userOne
	insertTestUser(t, repo, ctx, "user-2", "second@example.com", "OtherNick")

	duplicate := " nickname "
	_, err := service.UpdateProfile(ctx, "user-2", UpdateInput{Nickname: &duplicate})
	if err != ErrNicknameAlreadyExists {
		t.Fatalf("expected ErrNicknameAlreadyExists, got %v", err)
	}
}

func TestUpdateProfileAllowsSameNicknameForSameUser(t *testing.T) {
	service, repo := newTestService(t)
	ctx := context.Background()

	insertTestUser(t, repo, ctx, "user-1", "first@example.com", "Nickname")

	sameNickname := " nickname "
	updated, err := service.UpdateProfile(ctx, "user-1", UpdateInput{Nickname: &sameNickname})
	if err != nil {
		t.Fatalf("update profile failed: %v", err)
	}
	if updated.Nickname != "nickname" {
		t.Fatalf("expected trimmed nickname, got %q", updated.Nickname)
	}
}

func newTestService(t *testing.T) (*Service, *Repository) {
	t.Helper()

	db, err := sqlite.New("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrationsDir := filepath.Join("..", "db", "migrations", "sqlite")
	if err := sqlite.RunMigrations(db, migrationsDir); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	repo := NewRepository(db)
	return NewService(repo), repo
}

func insertTestUser(t *testing.T, repo *Repository, ctx context.Context, id, email, nickname string) User {
	t.Helper()

	now := time.Now().UTC()
	_, err := repo.db.ExecContext(ctx, `
		INSERT INTO users (
			id, email, password_hash, first_name, last_name, date_of_birth,
			nickname, profile_visibility, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, id, email, "hash", "Test", "User", now, nickname, "public", now, now)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	user, err := repo.GetUserByID(ctx, id)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	return user
}
