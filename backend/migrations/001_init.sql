-- Create posts table
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

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_posts_event_name ON posts(event_name);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_event_created ON posts(event_name, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_ip_hash_created ON posts(ip_hash, created_at);