package sessionauth

import (
	"context"
	"net/http"
	"social-network/backend/pkg/response"
)

const CookieName = "session_id"

type UserIDResolver func(ctx context.Context, sessionID string) (string, error)

func SessionIDFromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func RequireUserID(w http.ResponseWriter, r *http.Request, resolve UserIDResolver) (string, bool) {
	sessionID, err := SessionIDFromRequest(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "not authenticated")
		return "", false
	}

	userID, err := resolve(r.Context(), sessionID)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "not authenticated")
		return "", false
	}

	return userID, true
}
