package main

import "time"

type User struct {
	ID       string
	Name     string
	Username string
	Email    string
}

type Change struct {
	ChangeID  string
	Title     string
	Status    string // "open" / "merged" / "closed" / "draft"
	Author    User
	Project   string
	Branch    string
	Created   time.Time
	Updated   time.Time
	Mergeable bool
}

type Backend interface {
	Login()
	Logout()
	GetCurrentUser() (*User, error)
	GetChanges() ([]Change, error)
}
