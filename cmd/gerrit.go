package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
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

func (c *GerritClient) decodeResponse(resp *http.Response, v any) error {
	reader := bufio.NewReader(resp.Body)
	reader.ReadString('\n')
	return json.NewDecoder(reader).Decode(v)
}

func NewGerritClient() *GerritClient {
	host := os.Getenv(HostEnv)
	if host == "" {
		fmt.Printf("Environment variable %s cannot be empty\n", HostEnv)
		os.Exit(1)
	}

	username := os.Getenv(UsernameEnv)
	if username == "" {
		fmt.Printf("Environment variable %s cannot be empty\n", UsernameEnv)
		os.Exit(1)
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

type gerritAccountInfo struct {
	AccountID       int      `json:"_account_id"`
	Name            string   `json:"name"`
	DisplayName     string   `json:"display_name"`
	Email           string   `json:"email"`
	SecondaryEmails []string `json:"secondary_emails"`
	Username        string   `json:"username"`
}

func (a *gerritAccountInfo) ToUser() User {
	return User{
		ID:       strconv.Itoa(a.AccountID),
		Name:     a.Name,
		Username: a.Username,
		Email:    a.Email,
	}
}

func parseGerritTime(value string) time.Time {
	parsed, err := time.Parse("2006-01-02 15:04:05.999999999", value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

type gerritChange struct {
	ID                     string                                    `json:"id"`
	TripletID              string                                    `json:"triplet_id"`
	Project                string                                    `json:"project"`
	Branch                 string                                    `json:"branch"`
	FullBranch             string                                    `json:"full_branch"`
	Topic                  string                                    `json:"topic"`
	AttentionSet           map[string]json.RawMessage                `json:"attention_set"`
	RemovedAttentionSet    map[string]json.RawMessage                `json:"removed_from_attention_set"`
	Hashtags               []string                                  `json:"hashtags"`
	CustomKeyedValues      map[string]string                         `json:"custom_keyed_values"`
	ChangeID               string                                    `json:"change_id"`
	Subject                string                                    `json:"subject"`
	Status                 string                                    `json:"status"`
	Created                string                                    `json:"created"`
	Updated                string                                    `json:"updated"`
	Submitted              string                                    `json:"submitted"`
	Submitter              *gerritAccountInfo                        `json:"submitter"`
	Starred                bool                                      `json:"starred"`
	Reviewed               bool                                      `json:"reviewed"`
	SubmitType             string                                    `json:"submit_type"`
	Mergeable              *bool                                     `json:"mergeable"`
	Submittable            *bool                                     `json:"submittable"`
	Insertions             *int                                      `json:"insertions"`
	Deletions              *int                                      `json:"deletions"`
	TotalCommentCount      int                                       `json:"total_comment_count"`
	UnresolvedCommentCount int                                       `json:"unresolved_comment_count"`
	Number                 int                                       `json:"_number"`
	VirtualIDNumber        int                                       `json:"virtual_id_number"`
	Owner                  gerritAccountInfo                         `json:"owner"`
	Actions                map[string]json.RawMessage                `json:"actions"`
	SubmitRecords          []json.RawMessage                         `json:"submit_records"`
	Requirements           []json.RawMessage                         `json:"requirements"`
	SubmitRequirements     []json.RawMessage                         `json:"submit_requirements"`
	Labels                 map[string]json.RawMessage                `json:"labels"`
	PermittedLabels        map[string][]string                       `json:"permitted_labels"`
	RemovableLabels        map[string]map[string][]gerritAccountInfo `json:"removable_labels"`
	RemovableReviewers     []gerritAccountInfo                       `json:"removable_reviewers"`
	Reviewers              map[string][]gerritAccountInfo            `json:"reviewers"`
	PendingReviewers       map[string][]gerritAccountInfo            `json:"pending_reviewers"`
	ReviewerUpdates        []json.RawMessage                         `json:"reviewer_updates"`
	Messages               []json.RawMessage                         `json:"messages"`
	CurrentRevisionNumber  int                                       `json:"current_revision_number"`
	CurrentRevision        string                                    `json:"current_revision"`
	Revisions              map[string]json.RawMessage                `json:"revisions"`
	MetaRevID              string                                    `json:"meta_rev_id"`
	TrackingIDs            []json.RawMessage                         `json:"tracking_ids"`
	MoreChanges            bool                                      `json:"_more_changes"`
	Problems               []json.RawMessage                         `json:"problems"`
	IsPrivate              bool                                      `json:"is_private"`
	WorkInProgress         bool                                      `json:"work_in_progress"`
	HasReviewStarted       bool                                      `json:"has_review_started"`
	RevertOf               int                                       `json:"revert_of"`
	SubmissionID           string                                    `json:"submission_id"`
	CherryPickOfChange     int                                       `json:"cherry_pick_of_change"`
	CherryPickOfPatchSet   int                                       `json:"cherry_pick_of_patch_set"`
	ContainsGitConflicts   bool                                      `json:"contains_git_conflicts"`
}

func (c *GerritClient) GetCurrentUser() (*User, error) {
	url := "https://" + c.host + "/a/accounts/self"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
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

	var accountInfo gerritAccountInfo

	err = c.decodeResponse(resp, &accountInfo)
	if err != nil {
		return nil, err
	}

	user := accountInfo.ToUser()
	return &user, nil
}

func (c *GerritClient) Login() {
	if len(flag.Args()) != 2 {
		fmt.Println("Usage: revtui login [password]")
		os.Exit(1)
	}

	password := flag.Arg(1)

	err := SavePassword(c.host, c.username, password)
	if err != nil {
		fmt.Printf("Error saving password to OS keyring: %s\n", err.Error())
		os.Exit(1)
	}

	c.password = password

	accountInfo, err := c.GetCurrentUser()
	if err != nil {
		fmt.Printf("Error logging in: %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Logged in as %s\n", accountInfo.Name)
}

func (c *GerritClient) Logout() {
	err := DeletePasswordFor(c.host, c.username)
	if err != nil {
		fmt.Printf("Error deleting password from OS keyring: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Println("Logged out")
}

func (c *GerritClient) GetChanges() ([]Change, error) {
	url := "https://" + c.host + "/a/changes/?o=DETAILED_ACCOUNTS"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
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

	var gerritChanges []gerritChange

	err = c.decodeResponse(resp, &gerritChanges)
	if err != nil {
		return nil, err
	}

	changes := make([]Change, 0, len(gerritChanges))
	for _, change := range gerritChanges {
		mergeable := false
		if change.Mergeable != nil {
			mergeable = *change.Mergeable
		}

		changes = append(changes, Change{
			ChangeID:  change.ChangeID,
			Title:     change.Subject,
			Status:    change.Status,
			Author:    change.Owner.ToUser(),
			Project:   change.Project,
			Branch:    change.Branch,
			Created:   parseGerritTime(change.Created),
			Updated:   parseGerritTime(change.Updated),
			Mergeable: mergeable,
		})
	}

	return changes, nil
}
