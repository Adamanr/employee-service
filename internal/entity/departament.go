package entity

import "time"

type Department struct {
	ID          uint64    `json:"id,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ParentID    *uint64   `json:"parent_id"`
	HeadID      *uint64   `json:"head_id"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}
