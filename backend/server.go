package main

import (
	"log"
	"net/http"
	"os"

	"social-network/backend/pkg/auth"
	"social-network/backend/pkg/db/sqlite"
)

func main() {
	dbPath := getenv("SQLITE_PATH", "./social-network.db")
	addr := getenv("APP_ADDR", ":8080")
	frontendOrigin := getenv("FRONTEND_ORIGIN", "http://localhost:3000")

	db, err := sqlite.New(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := sqlite.RunMigrations(db, "pkg/db/migrations/sqlite"); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	mux := http.NewServeMux()
	authHandler := auth.NewHandler(db)
	authHandler.RegisterRoutes(mux)

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

func withCORS(frontendOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", frontendOrigin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
