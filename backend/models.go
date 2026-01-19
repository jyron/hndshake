package main

import "time"

type Post struct {
	ID        int       `json:"id"`
	EventName string    `json:"event_name"`
	Content   string    `json:"content"`
	AgeRange  string    `json:"age_range"`
	Gender    string    `json:"gender"`
	Location  string    `json:"location"`
	CreatedAt time.Time `json:"created_at"`
}

type CreatePostRequest struct {
	EventName string `json:"event_name"`
	Content   string `json:"content"`
	AgeRange  string `json:"age_range"`
	Gender    string `json:"gender"`
	Location  string `json:"location"`
}