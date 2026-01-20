package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	conn *sql.DB
}

func NewDB(databaseURL string) (*DB, error) {
	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(2)
	conn.SetConnMaxLifetime(time.Hour)
	conn.SetConnMaxIdleTime(30 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to database")

	return &DB{conn: conn}, nil
}

func (db *DB) Close() {
	db.conn.Close()
}

// CreatePost inserts a new post into the database
func (db *DB) CreatePost(ctx context.Context, req CreatePostRequest, ipHash string) (*Post, error) {
	query := `
		INSERT INTO posts (event_name, content, age_range, gender, location, ip_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, event_name, content, age_range, gender, location, created_at
	`

	var post Post
	err := db.conn.QueryRowContext(
		ctx,
		query,
		req.EventName,
		req.Content,
		req.AgeRange,
		req.Gender,
		req.Location,
		ipHash,
	).Scan(
		&post.ID,
		&post.EventName,
		&post.Content,
		&post.AgeRange,
		&post.Gender,
		&post.Location,
		&post.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	return &post, nil
}

// GetPosts retrieves posts, optionally filtered by event
func (db *DB) GetPosts(ctx context.Context, eventFilter string, limit int, offset int) ([]Post, error) {
	var query string
	var args []interface{}

	if eventFilter != "" {
		query = `
			SELECT id, event_name, content, age_range, gender, location, created_at
			FROM posts
			WHERE event_name = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{eventFilter, limit, offset}
	} else {
		query = `
			SELECT id, event_name, content, age_range, gender, location, created_at
			FROM posts
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset}
	}

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query posts: %w", err)
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(
			&post.ID,
			&post.EventName,
			&post.Content,
			&post.AgeRange,
			&post.Gender,
			&post.Location,
			&post.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating posts: %w", err)
	}

	return posts, nil
}

// GetEvents retrieves all unique event names ordered by most recent post
func (db *DB) GetEvents(ctx context.Context) ([]string, error) {
	query := `
		SELECT event_name
		FROM posts
		GROUP BY event_name
		ORDER BY MAX(created_at) DESC
	`

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []string
	for rows.Next() {
		var event string
		if err := rows.Scan(&event); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

// GetPostCountByIPInWindow checks how many posts an IP has made in the time window
func (db *DB) GetPostCountByIPInWindow(ctx context.Context, ipHash string, windowMinutes int) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM posts
		WHERE ip_hash = $1
		AND created_at > NOW() - INTERVAL '1 minute' * $2
	`

	var count int
	err := db.conn.QueryRowContext(ctx, query, ipHash, windowMinutes).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count posts: %w", err)
	}

	return count, nil
}

func runMigrations(db *DB) {
    migration := `
        CREATE TABLE IF NOT EXISTS posts (
            id SERIAL PRIMARY KEY,
            event_name VARCHAR(200) NOT NULL,
            content TEXT NOT NULL,
            age_range VARCHAR(20) NOT NULL,
            gender VARCHAR(20),
            location VARCHAR(200) NOT NULL,
            ip_hash VARCHAR(64) NOT NULL,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );
        
        CREATE INDEX IF NOT EXISTS idx_posts_event_name ON posts(event_name);
        CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
        CREATE INDEX IF NOT EXISTS idx_posts_event_created ON posts(event_name, created_at DESC);
        CREATE INDEX IF NOT EXISTS idx_posts_ip_hash_created ON posts(ip_hash, created_at);
    `
    
    _, err := db.conn.Exec(migration)
    if err != nil {
        log.Fatalf("Failed to run migrations: %v", err)
    }
    log.Println("Migrations completed successfully")
}