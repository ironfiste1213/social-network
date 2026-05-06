// backend/server.go — complete replacement
package main

import (
	"log"
	"net/http"
	"os"

	"social-network/backend/pkg/auth"
	"social-network/backend/pkg/chat"
	"social-network/backend/pkg/comments"
	"social-network/backend/pkg/db/sqlite"
	"social-network/backend/pkg/events"
	"social-network/backend/pkg/followers"
	"social-network/backend/pkg/groups"
	"social-network/backend/pkg/notifications"
	"social-network/backend/pkg/posts"
	"social-network/backend/pkg/users"
)

func main() {
	dbPath := getenv("SQLITE_PATH", "./social-network.db")
	addr := getenv("APP_ADDR", ":8080")
	frontendOrigin := getenv("FRONTEND_ORIGIN", "http://localhost:3000")
	uploadDir := getenv("UPLOAD_DIR", "./uploads")

	db, err := sqlite.New(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := sqlite.RunMigrations(db, "pkg/db/migrations/sqlite"); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	mux := http.NewServeMux()

	// ── Auth ──────────────────────────────────────────────
	authHandler := auth.NewHandler(db)
	authHandler.RegisterRoutes(mux)

	// ── Users ─────────────────────────────────────────────
	usersHandler := users.NewHandler(db, uploadDir)
	usersHandler.RegisterRoutes(mux)

	// ── Notifications ─────────────────────────────────────
	// Must be created before followers/groups/events
	// so we can pass it as a dependency
	notifService := notifications.NewServiceFromDB(db)
	notifHandler := notifications.NewHandlerWithService(db, notifService)
	notifHandler.RegisterRoutes(mux)

	// ── Followers ─────────────────────────────────────────
	followersHandler := followers.NewHandler(db, notifService)
	followersHandler.RegisterRoutes(mux)
	usersHandler.SetFollowersHandler(followersHandler)

	// ── Posts ─────────────────────────────────────────────
	postsHandler := posts.NewHandler(db, uploadDir)
	postsHandler.RegisterRoutes(mux)
	usersHandler.SetPostsHandler(postsHandler)
	mux.HandleFunc("/posts/my-followers", postsHandler.GetMyFollowers)

	// ── Comments ──────────────────────────────────────────
	commentsHandler := comments.NewHandler(db, uploadDir)
	postsHandler.SetCommentsHandler(commentsHandler)

	// ── Events ────────────────────────────────────────────
	eventsHandler := events.NewHandler(db, notifService)

	// ── Groups ────────────────────────────────────────────
	groupsHandler := groups.NewHandler(db, notifService)
	groupsHandler.RegisterRoutes(mux)
	groupsHandler.SetEventsHandler(eventsHandler)
	groupsHandler.SetPostsHandler(postsHandler)

	// ── Chat ──────────────────────────────────────────────
	hub := chat.NewHub()
	go hub.Run() // must run in background goroutine
	chatHandler := chat.NewHandler(db, hub)
	chatHandler.RegisterRoutes(mux)

	// ── Static uploads ────────────────────────────────────
	mux.Handle("/uploads/", users.ServeUploads(uploadDir))

	// ── Health ────────────────────────────────────────────
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("backend listening on %s", addr)
	if err := http.ListenAndServe(addr, withCORS(frontendOrigin, mux)); err != nil {
		log.Fatalf("listen and serve: %v", err)
	}
}

func withCORS(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
