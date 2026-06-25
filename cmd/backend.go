package main

import "time"

type User struct {
	ID       string
	Name     string
	Username string
	Email    string
}

type ReviewStatus string

const (
	ReviewStatusUnknown        ReviewStatus = "unknown"
	ReviewStatusNotReady       ReviewStatus = "not_ready"
	ReviewStatusReadyForReview ReviewStatus = "ready_for_review"
	ReviewStatusReviewed       ReviewStatus = "reviewed"
	ReviewStatusVerified       ReviewStatus = "verified"
	ReviewStatusBlocked        ReviewStatus = "blocked"
)

type ReviewSummary struct {
	Primary  ReviewStatus
	Statuses []ReviewStatus
}

type ChangeFlags struct {
	HasConflicts     bool
	IsWorkInProgress bool
}

type Change struct {
	ChangeID string
	Title    string
	Status   string // "open" / "merged" / "closed" / "draft"
	Review   ReviewSummary
	Flags    ChangeFlags
	Author   User
	Project  string
	Branch   string
	Created  time.Time
	Updated  time.Time
}

type Backend interface {
	Login()
	Logout()
	GetCurrentUser() (*User, error)
	GetChanges() ([]Change, error)
	Checkout(Change) error
}
