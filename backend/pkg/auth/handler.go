package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"social-network/backend/pkg/response"
	"social-network/backend/pkg/sessionauth"
)

type Handler struct {
	service *Service
}

func NewHandler(db *sql.DB) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)

	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/auth/register", h.handleRegister)
	mux.HandleFunc("/auth/login", h.handleLogin)
	mux.HandleFunc("/auth/logout", h.handleLogout)
	mux.HandleFunc("/auth/me", h.handleMe)
}

// POST /auth/register
func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var input RegisterInput
	if err := decodeJSON(r, &input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}

	fmt.Println("[HANDLER] Register request started for email:", input.Email)

	user, session, err := h.service.Register(r.Context(), input)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	fmt.Println("[HANDLER] Register response: success")
	setSessionCookie(w, session)
	response.JSON(w, http.StatusCreated, map[string]any{
		"message": "registration successful",
		"user":    user.Safe(),
	})
}

// POST /auth/login
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var input LoginInput
	if err := decodeJSON(r, &input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}

	fmt.Println("[HANDLER] Login request started for email:", input.Email)

	user, session, err := h.service.Login(r.Context(), input)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	fmt.Println("[HANDLER] Login response: success")
	setSessionCookie(w, session)
	response.JSON(w, http.StatusOK, map[string]any{
		"message": "login successful",
		"user":    user.Safe(),
	})
}

// POST /auth/logout
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sessionID, err := sessionauth.SessionIDFromRequest(r)
	if err == nil {
		if err := h.service.Logout(r.Context(), sessionID); err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to logout")
			return
		}
	}

	clearSessionCookie(w)
	response.JSON(w, http.StatusOK, map[string]string{
		"message": "logout successful",
	})
}

// GET /auth/me
func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sessionID, err := sessionauth.SessionIDFromRequest(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.service.CurrentUser(r.Context(), sessionID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get current user")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"user": user.Safe(),
	})
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		response.Error(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, ErrInvalidCredentials):
		response.Error(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, ErrEmailAlreadyExists):
		response.Error(w, http.StatusConflict, "email already exists")
	case errors.Is(err, ErrNicknameAlreadyExists):
		response.Error(w, http.StatusConflict, "nickname already exists")
	default:
		response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	return decoder.Decode(target)
}

func setSessionCookie(w http.ResponseWriter, session Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionauth.CookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
		MaxAge:   int(time.Until(session.ExpiresAt).Seconds()),
	})
	fmt.Println("ok!")
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionauth.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
