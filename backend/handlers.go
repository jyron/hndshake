package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	db *DB
}

func NewHandler(db *DB) *Handler {
	return &Handler{db: db}
}

// CreatePost handles POST /api/posts
func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	var req CreatePostRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validateCreatePostRequest(req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get IP hash from context (set by rate limiter)
	ipHash := IPHashFromContext(r.Context())
	if ipHash == "" {
		ipHash = computeIPHash(r)
	}

	// Create post
	post, err := h.db.CreatePost(r.Context(), req, ipHash)
	if err != nil {
		log.Printf("Error creating post: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to create post")
		return
	}

	respondWithJSON(w, http.StatusCreated, post)
}

// GetPosts handles GET /api/posts
func (h *Handler) GetPosts(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	eventFilter := r.URL.Query().Get("event")

	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offsetStr := r.URL.Query().Get("offset")
	offset := 0 // default
	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Get posts
	posts, err := h.db.GetPosts(r.Context(), eventFilter, limit, offset)
	if err != nil {
		log.Printf("Error getting posts: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve posts")
		return
	}

	// Return empty array instead of null if no posts
	if posts == nil {
		posts = []Post{}
	}

	respondWithJSON(w, http.StatusOK, posts)
}

// GetEvents handles GET /api/events
func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.db.GetEvents(r.Context())
	if err != nil {
		log.Printf("Error getting events: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve events")
		return
	}

	// Return empty array instead of null if no events
	if events == nil {
		events = []string{}
	}

	respondWithJSON(w, http.StatusOK, events)
}

// Helper functions

func validateCreatePostRequest(req CreatePostRequest) error {
	req.EventName = strings.TrimSpace(req.EventName)
	req.Content = strings.TrimSpace(req.Content)
	req.Location = strings.TrimSpace(req.Location)

	if req.EventName == "" {
		return &ValidationError{"event_name is required"}
	}
	if len(req.EventName) > 200 {
		return &ValidationError{"event_name must be 200 characters or less"}
	}

	if req.Content == "" {
		return &ValidationError{"content is required"}
	}
	if len(req.Content) > 5000 {
		return &ValidationError{"content must be 5000 characters or less"}
	}

	// Age must be between 1 and 120
	if req.Age < 1 || req.Age > 120 {
		return &ValidationError{"age must be between 1 and 120"}
	}

	if req.Location == "" {
		return &ValidationError{"location is required"}
	}
	if len(req.Location) > 200 {
		return &ValidationError{"location must be 200 characters or less"}
	}

	// Gender is optional, but validate if provided
	if req.Gender != "" && len(req.Gender) > 20 {
		return &ValidationError{"gender must be 20 characters or less"}
	}

	return nil
}

func computeIPHash(r *http.Request) string {
	ip := r.RemoteAddr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}
	return hashIP(ip)
}

func respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func respondWithError(w http.ResponseWriter, status int, message string) {
	respondWithJSON(w, status, map[string]string{"error": message})
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}