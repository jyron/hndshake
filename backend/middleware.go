package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

type RateLimiter struct {
	db             *DB
	requestLimit   int
	windowMinutes  int
}

func NewRateLimiter(db *DB, requestLimit, windowMinutes int) *RateLimiter {
	return &RateLimiter{
		db:            db,
		requestLimit:  requestLimit,
		windowMinutes: windowMinutes,
	}
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only rate limit POST requests
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		ip := getIP(r)
		ipHash := hashIP(ip)

		count, err := rl.db.GetPostCountByIPInWindow(r.Context(), ipHash, rl.windowMinutes)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if count >= rl.requestLimit {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(fmt.Sprintf(`{"error":"Rate limit exceeded. Maximum %d posts per %d minutes."}`, rl.requestLimit, rl.windowMinutes)))
			return
		}

		// Store IP hash in context for use in handlers
		ctx := context.WithValue(r.Context(), ipHashKey, ipHash)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}
	return ip
}

func hashIP(ip string) string {
	hash := sha256.Sum256([]byte(ip + "living-timeline-salt"))
	return hex.EncodeToString(hash[:])
}

// Context key for IP hash
type contextKey string

const ipHashKey contextKey = "ipHash"

func IPHashFromContext(ctx context.Context) string {
	if ipHash, ok := ctx.Value(ipHashKey).(string); ok {
		return ipHash
	}
	return ""
}