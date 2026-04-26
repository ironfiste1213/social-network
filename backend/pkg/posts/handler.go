package posts

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"social-network/backend/pkg/response"
	"social-network/backend/pkg/sessionauth"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const maxImageSize = 10 << 20 // 10MB

type Handler struct {
	service   *Service
	uploadDir string
	comments  PostSubrouteHandler
}

type PostSubrouteHandler interface {
	HandlePostSubroute(w http.ResponseWriter, r *http.Request, postID, subpath string) bool
}

func NewHandler(db *sql.DB, uploadDir string) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)
	return &Handler{service: service, uploadDir: uploadDir}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/posts", h.handlePosts)
	mux.HandleFunc("/posts/", h.handlePostByID)
}

func (h *Handler) SetCommentsHandler(handler PostSubrouteHandler) {
	h.comments = handler
}

// POST /posts         — create
// GET  /posts         — feed
func (h *Handler) handlePosts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createPost(w, r)
	case http.MethodGet:
		h.getFeed(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// DELETE /posts/{id}
// POST   /posts/{id}/image
func (h *Handler) handlePostByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/posts/")
	parts := strings.SplitN(path, "/", 2)
	postID := parts[0]
	if postID == "" {
		http.NotFound(w, r)
		return
	}

	if len(parts) == 2 && h.comments != nil && h.comments.HandlePostSubroute(w, r, postID, parts[1]) {
		return
	}

	if len(parts) == 2 && parts[1] == "image" {
		h.uploadImage(w, r, postID)
		return
	}
	if r.Method == http.MethodDelete {
		h.deletePost(w, r, postID)
		return
	}
	response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
}

// GET /users/{id}/posts
func (h *Handler) HandleUserPostRoutes(w http.ResponseWriter, r *http.Request, authorID, sub string) bool {
	if sub != "posts" {
		return false
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return true
	}
	h.getUserPosts(w, r, authorID)
	return true
}

func (h *Handler) createPost(w http.ResponseWriter, r *http.Request) {
	authorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	var input CreatePostInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	post, err := h.service.CreatePost(r.Context(), authorID, input)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, "invalid input: body required; selected_followers requires viewer_ids")
			return
		}
		if errors.Is(err, ErrForbidden) {
			response.Error(w, http.StatusForbidden, "you must be a group member to post in this group")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to create post")
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"post": post})
}

func (h *Handler) getFeed(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	limit, offset := parsePagination(r)
	posts, err := h.service.GetFeed(r.Context(), viewerID, limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to get feed")
		return
	}
	if posts == nil {
		posts = []Post{}
	}
	response.JSON(w, http.StatusOK, map[string]any{"posts": posts})
}

func (h *Handler) getUserPosts(w http.ResponseWriter, r *http.Request, authorID string) {
	viewerID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	limit, offset := parsePagination(r)
	posts, err := h.service.GetUserPosts(r.Context(), authorID, viewerID, limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to get posts")
		return
	}
	if posts == nil {
		posts = []Post{}
	}
	response.JSON(w, http.StatusOK, map[string]any{"posts": posts})
}

func (h *Handler) deletePost(w http.ResponseWriter, r *http.Request, postID string) {
	requesterID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	if err := h.service.DeletePost(r.Context(), postID, requesterID); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "post not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to delete post")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *Handler) uploadImage(w http.ResponseWriter, r *http.Request, postID string) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	authorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxImageSize)
	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		response.Error(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "missing image file")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true}
	if !allowed[ext] {
		response.Error(w, http.StatusBadRequest, "only jpg, png, gif allowed")
		return
	}

	filename := uuid.NewString() + ext
	dest := filepath.Join(h.uploadDir, filename)
	out, err := os.Create(dest)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to write file")
		return
	}

	imagePath := "/uploads/" + filename
	if err := h.service.UpdateImagePath(r.Context(), postID, authorID, imagePath); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to update post image")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"image_path": imagePath})
}

// GET /posts/my-followers  — list of your followers for the privacy picker
func (h *Handler) GetMyFollowers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	followers, err := h.service.GetFollowersOfUser(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to get followers")
		return
	}
	if followers == nil {
		followers = []FollowerSummary{}
	}
	response.JSON(w, http.StatusOK, map[string]any{"followers": followers})
}

// --- helpers ---

func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) (string, bool) {
	return sessionauth.RequireUserID(w, r, h.service.CurrentUserID)
}

func parsePagination(r *http.Request) (limit, offset int) {
	limit = 20
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}
