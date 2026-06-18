package main

import (
	"flag"
	"fmt"
	"os"
)

func printUser(a *User) {
	fmt.Printf("%-11s %s\n%-11s %s\n%-11s %s\n%-11s %s\n", "Account ID:", a.ID, "Name:", a.Name, "Email:", a.Email, "Username:", a.Username)
}

func userDisplayName(a *User) string {
	if a.Name != "" {
		return a.Name
	}
	if a.Username != "" {
		return a.Username
	}
	if a.Email != "" {
		return a.Email
	}
	if a.ID != "" {
		return a.ID
	}
	return ""
}

func handleLogin(client Backend) {
	client.Login()
}

func handleLogout(client Backend) {
	client.Logout()
}

func handleGetMe(client Backend) {
	user, err := client.GetCurrentUser()
	if err != nil {
		fmt.Printf("Error getting account info: %s\n", err.Error())
		os.Exit(1)
	}

	printUser(user)
}

func handleNuke() {
	err := DeleteAllPasswords()
	if err != nil {
		fmt.Printf("Error nuking passwords: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Println("All passwords removed from OS keyring")
}

const (
	ChangeIDField = "Change ID"
	SubjectField  = "Subject"
	OwnerField    = "Owner"
)

func handleChanges(client Backend) {
	changes, err := client.GetChanges()
	if err != nil {
		fmt.Printf("Error retrieving changes: %s\n", err.Error())
		os.Exit(1)
	}

	longestChangeID := len(ChangeIDField)
	longestSubject := len(SubjectField)
	longestOwner := len(OwnerField)

	for _, change := range changes {
		longestChangeID = max(longestChangeID, len(change.ChangeID))
		longestSubject = max(longestSubject, len(change.Title))
		longestOwner = max(longestOwner, len(userDisplayName(&change.Author)))
	}

	fmt.Printf("%-*s %-*s %-*s\n", longestChangeID, ChangeIDField, longestSubject, SubjectField, longestOwner, OwnerField)

	for _, change := range changes {
		fmt.Printf("%-*s %-*s %-*s\n", longestChangeID, change.ChangeID, longestSubject, change.Title, longestOwner, userDisplayName(&change.Author))
	}
}

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Println("Subcommand required [inbox|changes|login|logout|nuke|me]")
		os.Exit(1)
	}

	switch cmd := flag.Arg(0); cmd {
	case "inbox":
		fmt.Println("Doing inbox things")
	case "changes":
		client := NewGerritClient()
		handleChanges(client)
	case "login":
		client := NewGerritClient()
		handleLogin(client)
	case "logout":
		client := NewGerritClient()
		handleLogout(client)
	case "nuke":
		handleNuke()
	case "me":
		client := NewGerritClient()
		handleGetMe(client)
	default:
		fmt.Printf("Unsupported subcommand: %s\n", cmd)
		os.Exit(1)
	}
}
