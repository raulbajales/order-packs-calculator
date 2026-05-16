package models

import "time"

// Single pack size record in the database.
type Pack struct {
	ID        int
	Size      int
	CreatedAt time.Time
}
