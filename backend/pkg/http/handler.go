package http

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

type Handler struct {
	service *Service
}

func NewHandler(db *sql.DB) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)
	return &Handler{service: service}
}

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Get the session cookie
		cookie, err := r.Cookie("session_token")
		if err != nil {
			if err == http.ErrNoCookie {
				// No session cookie, user is not authenticated for this request
				RespondWithError(w, http.StatusUnauthorized, "Authentication required: No session cookie")
				return
			}
			// Other cookie-related errors
			// log.Printf("[middleware.go:AuthMiddleware] AuthMiddleware: Error getting cookie: %v", err)
			RespondWithError(w, http.StatusBadRequest, "Bad request")
			return
		}

		sessionToken := cookie.Value

		// 2. Validate the session token against the database
		session, err := h.service.GetSessionByToken(sessionToken)
		if err != nil {
			// This covers cases where the session is not found or other DB errors
			// log.Printf("[middleware.go:AuthMiddleware] Error retrieving session: %v", err)
			ClearSessionCookie(w) // Clear potentially invalid cookie
			RespondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		if session == nil {
			// Session token not found in DB
			// log.Printf("[middleware.go:AuthMiddleware] AuthMiddleware: Session token not found in database: %s", sessionToken)
			ClearSessionCookie(w)
			RespondWithError(w, http.StatusUnauthorized, "Invalid session")
			return
		}

		// Check if the session has expired
		if session.Expiry.Before(time.Now()) {
			// log.Printf("[middleware.go:AuthMiddleware] Session expired for user ID: %d, token: %s", session.UserID, sessionToken)
			_ = h.service.repo.DeleteSession(sessionToken) // Clean up expired session from DB
			ClearSessionCookie(w)
			RespondWithError(w, http.StatusUnauthorized, "Session expired")
			return
		}

		// 3. Retrieve the user associated with the session
		user, err := h.service.repo.GetUserBySessionID(r.Context(), session.ID)
		if err != nil {
			// log.Printf("[middleware.go:AuthMiddleware] AuthMiddleware: Error retrieving user for session %s (UserID: %d): %v", sessionToken, session.UserID, err)
			ClearSessionCookie(w)
			RespondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		if user == nil {
			// User associated with session not found (e.g., user deleted but session remains)
			// log.Printf("[middleware.go:AuthMiddleware] User (ID: %d) not found for session %s", session.UserID, sessionToken)
			_ = h.service.repo.DeleteSession(sessionToken) // Invalidate the session as it points to a non-existent user, and clear the client cookie
			ClearSessionCookie(w)
			RespondWithError(w, http.StatusUnauthorized, "User not found for session")
			return
		}

		// 4. Add the user to the request context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		r = r.WithContext(ctx)

		// log.Printf("[middleware.go:AuthMiddleware] AuthMiddleware: User %s (ID: %d) authenticated successfully. Proceeding to handler.", user.Nickname, user.ID)

		// 5. Call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}

// ClearSessionCookie removes the session cookie from the client's browser.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // This tells the browser to delete the cookie
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	})
}
