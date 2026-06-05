package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

var (
	HostEnv       = "GERRIT_HOST"
	UsernameEnv   = "GERRIT_USER"
	SkipVerifyTLS = "GERRIT_SKIP_VERIFY_TLS"
)

type GerritClient struct {
	host       string
	username   string
	password   string
	httpClient *http.Client
}

func (c *GerritClient) DecodeResponse(resp *http.Response, v any) error {
	reader := bufio.NewReader(resp.Body)
	reader.ReadString('\n')
	return json.NewDecoder(reader).Decode(v)
}

func NewGerritClient() *GerritClient {
	host := os.Getenv(HostEnv)
	if host == "" {
		log.Fatalf("Environment variable %s cannot be empty", HostEnv)
	}

	username := os.Getenv(UsernameEnv)
	if username == "" {
		log.Fatalf("Environment variable %s cannot be empty", UsernameEnv)
	}

	password, _ := GetPassword(host, username) // Ignore error.  It's okay if password is empty

	return &GerritClient{
		host,
		username,
		password,
		&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: os.Getenv(SkipVerifyTLS) == "1",
				},
			},
		},
	}
}

type AccountInfo struct {
	AccountID int    `json:"_account_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Username  string `json:"username"`
}

func (a *AccountInfo) Print() {
	fmt.Printf("%-11s %d\n%-11s %s\n%-11s %s\n%-11s %s\n", "Account ID:", a.AccountID, "Name:", a.Name, "Email:", a.Email, "Username:", a.Username)
}

func getAccountInfo(client *GerritClient) (*AccountInfo, error) {
	url := "https://" + client.host + "/a/accounts/self"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(client.username, client.password)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(responseBody))
	}

	var accountInfo AccountInfo

	err = client.DecodeResponse(resp, &accountInfo)
	if err != nil {
		return nil, err
	}

	return &accountInfo, nil
}

func handleLogin(client *GerritClient) {
	if len(flag.Args()) != 2 {
		fmt.Println("Usage: gerritui auth [password]")
		os.Exit(1)
	}

	password := flag.Arg(1)

	err := SavePassword(client.host, client.username, password)
	if err != nil {
		log.Fatalf("Error saving password to OS keyring: %s", err.Error())
	}

	client.password = password

	accountInfo, err := getAccountInfo(client)
	if err != nil {
		log.Fatalf("Error logging in: %s", err.Error())
	}
	fmt.Printf("Logged in as %s\n", accountInfo.Name)
}

func handleLogout(client *GerritClient) {
	err := DeletePasswordFor(client.host, client.username)
	if err != nil {
		log.Fatalf("Error deleting password from OS keyring: %s", err.Error())
	}

	fmt.Println("Logged out")
}

func handleGetMe(client *GerritClient) {
	accountInfo, err := getAccountInfo(client)
	if err != nil {
		log.Fatalf("Error getting account info: %s", err.Error())
	}
	accountInfo.Print()
}

func handleNuke() {
	err := DeleteAllPasswords()
	if err != nil {
		log.Fatalf("Error nuking passwords: %s", err.Error())
	}

	fmt.Println("All passwords removed from OS keyring")
}

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatalln("Subcommand required [inbox|login|me]")
	}

	switch cmd := flag.Arg(0); cmd {
	case "inbox":
		fmt.Println("Doing inbox things")
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
		log.Fatalf("Unsupported subcommand: %s", cmd)
	}
}
