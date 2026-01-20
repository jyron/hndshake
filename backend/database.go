package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
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
		INSERT INTO posts (event_name, content, age, gender, location, ip_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, event_name, content, age, gender, location, created_at
	`

	var post Post
	err := db.conn.QueryRowContext(
		ctx,
		query,
		req.EventName,
		req.Content,
		req.Age,
		req.Gender,
		req.Location,
		ipHash,
	).Scan(
		&post.ID,
		&post.EventName,
		&post.Content,
		&post.Age,
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
			SELECT id, event_name, content, age, gender, location, created_at
			FROM posts
			WHERE event_name = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{eventFilter, limit, offset}
	} else {
		query = `
			SELECT id, event_name, content, age, gender, location, created_at
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
			&post.Age,
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

// runMigrations executes all pending database migrations in order.
// It reads migration files from the migrations/ folder and tracks which have been run.
func runMigrations(db *DB) {
	// Create schema_migrations table to track applied migrations
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create schema_migrations table: %v", err)
	}

	// Read migration files from migrations/ directory
	migrationFiles, err := os.ReadDir("migrations")
	if err != nil {
		log.Fatalf("Failed to read migrations directory: %v", err)
	}

	// Sort migration files by name to ensure order
	var sortedFiles []string
	for _, file := range migrationFiles {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			sortedFiles = append(sortedFiles, file.Name())
		}
	}
	
	// Files are already sorted alphabetically (001_, 002_, etc.)
	for _, filename := range sortedFiles {
		// Extract version from filename (e.g., "001_init.sql" -> "001_init")
		version := strings.TrimSuffix(filename, ".sql")

		// Check if migration has been applied
		var exists bool
		err := db.conn.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
			version,
		).Scan(&exists)
		if err != nil {
			log.Fatalf("Failed to check migration status for %s: %v", version, err)
		}

		if exists {
			log.Printf("Migration %s already applied, skipping", version)
			continue
		}

		// Read migration file
		sqlBytes, err := os.ReadFile(fmt.Sprintf("migrations/%s", filename))
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v", filename, err)
		}

		// Run the migration
		log.Printf("Applying migration: %s", version)
		_, err = db.conn.Exec(string(sqlBytes))
		if err != nil {
			log.Fatalf("Failed to run migration %s: %v", version, err)
		}

		// Record the migration
		_, err = db.conn.Exec(
			"INSERT INTO schema_migrations (version) VALUES ($1)",
			version,
		)
		if err != nil {
			log.Fatalf("Failed to record migration %s: %v", version, err)
		}

		log.Printf("Migration %s completed successfully", version)
	}

	log.Println("All migrations completed successfully")
}