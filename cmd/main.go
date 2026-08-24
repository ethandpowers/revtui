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
	FlagsField    = ""
	ChangeIDField = "Change ID"
	ReviewField   = "Review"
	SubjectField  = "Subject"
	OwnerField    = "Owner"
)

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		client := NewGerritClient()
		renderTUI(client)
		os.Exit(0)
	}

	switch cmd := flag.Arg(0); cmd {
	case "changes":
		client := NewGerritClient()
		renderTUI(client)
		os.Exit(0)
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
		fmt.Println("Supported subcommands: [login|logout|nuke|me]")
		os.Exit(1)
	}
}
