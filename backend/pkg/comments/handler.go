package comments

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"social-network/backend/pkg/response"
	"social-network/backend/pkg/sessionauth"

	"github.com/google/uuid"
)

const maxImageSize = 10 << 20 // 10MB

type Handler struct {
	service   *Service
	uploadDir string
}

func NewHandler(db *sql.DB, uploadDir string) *Handler {
	repo := NewRepository(db)
	service := NewService(repo)
	return &Handler{service: service, uploadDir: uploadDir}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// /posts/{postID}/comments
	// /posts/{postID}/comments/{commentID}
	// /posts/{postID}/comments/{commentID}/image
	mux.HandleFunc("/posts/", h.route)
}

func (h *Handler) HandlePostSubroute(w http.ResponseWriter, r *http.Request, postID, subpath string) bool {
	if subpath != "comments" && !strings.HasPrefix(subpath, "comments/") {
		return false
	}
	h.routeWithPostID(w, r, postID, subpath)
	return true
}

func (h *Handler) route(w http.ResponseWriter, r *http.Request) {
	// Path examples:
	//   /posts/{postID}/comments
	//   /posts/{postID}/comments/{commentID}
	//   /posts/{postID}/comments/{commentID}/image

	path := strings.TrimPrefix(r.URL.Path, "/posts/")
	parts := strings.SplitN(path, "/", 4)
	// parts[0] = postID
	// parts[1] = "comments"
	// parts[2] = commentID (optional)
	// parts[3] = "image" (optional)

	if len(parts) < 2 || parts[1] != "comments" {
		http.NotFound(w, r)
		return
	}

	postID := parts[0]
	if postID == "" {
		http.NotFound(w, r)
		return
	}

	h.routeWithPostID(w, r, postID, strings.Join(parts[1:], "/"))
}

func (h *Handler) routeWithPostID(w http.ResponseWriter, r *http.Request, postID, subpath string) {
	parts := strings.SplitN(subpath, "/", 3)
	if len(parts) == 0 || parts[0] != "comments" {
		http.NotFound(w, r)
		return
	}

	// /posts/{postID}/comments
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			h.listComments(w, r, postID)
		case http.MethodPost:
			h.createComment(w, r, postID)
		default:
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	commentID := parts[1]
	if commentID == "" {
		http.NotFound(w, r)
		return
	}

	// /posts/{postID}/comments/{commentID}/image
	if len(parts) == 3 && parts[2] == "image" {
		if r.Method != http.MethodPost {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.uploadImage(w, r, postID, commentID)
		return
	}

	// /posts/{postID}/comments/{commentID}
	if r.Method == http.MethodDelete {
		h.deleteComment(w, r, commentID)
		return
	}

	response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
}

// POST /posts/{postID}/comments
func (h *Handler) createComment(w http.ResponseWriter, r *http.Request, postID string) {
	authorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	var input CreateCommentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}

	comment, err := h.service.CreateComment(r.Context(), postID, authorID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(w, http.StatusBadRequest, "body is required")
		case errors.Is(err, ErrForbidden):
			response.Error(w, http.StatusForbidden, "cannot comment on this post")
		default:
			response.Error(w, http.StatusInternalServerError, "failed to create comment")
		}
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{"comment": comment})
}

// GET /posts/{postID}/comments
func (h *Handler) listComments(w http.ResponseWriter, r *http.Request, postID string) {
	viewerID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	comments, err := h.service.GetComments(r.Context(), postID, viewerID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			response.Error(w, http.StatusForbidden, "cannot view comments on this post")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get comments")
		return
	}

	if comments == nil {
		comments = []Comment{}
	}
	response.JSON(w, http.StatusOK, map[string]any{"comments": comments})
}

// DELETE /posts/{postID}/comments/{commentID}
func (h *Handler) deleteComment(w http.ResponseWriter, r *http.Request, commentID string) {
	requesterID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	if err := h.service.DeleteComment(r.Context(), commentID, requesterID); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "comment not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to delete comment")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// POST /posts/{postID}/comments/{commentID}/image
func (h *Handler) uploadImage(w http.ResponseWriter, r *http.Request, postID, commentID string) {
	authorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	if err := h.service.VerifyCommentPost(r.Context(), commentID, postID); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "comment not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to validate comment")
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
	if err := h.service.UpdateImagePath(r.Context(), commentID, authorID, imagePath); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to update comment image")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"image_path": imagePath})
}

func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) (string, bool) {
	return sessionauth.RequireUserID(w, r, h.service.CurrentUserID)
}
