package main

import (
	"errors"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

const MetadataSeparator = "---"

type Patch struct {
	Metadata CommitMetadata
	Files    []FileDiff
}

type CommitMetadata struct {
	Hash    string
	Author  User
	Date    time.Time
	Subject string
	Body    string
}

type FileDiff struct {
	OldPath string
	NewPath string
	Hunks   []Hunk
}

type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []DiffLine
}

type DiffLine struct {
	Type DiffLineType
	Text string
}

type DiffLineType int

const (
	DiffLineContext DiffLineType = iota
	DiffLineAdded
	DiffLineRemoved
)

func (p Patch) String() string {
	var s strings.Builder

	s.WriteString(p.Metadata.String())
	s.WriteString("\n")

	for _, file := range p.Files {
		s.WriteString(file.String())
		s.WriteString("\n")
	}

	return s.String()
}

func (m CommitMetadata) String() string {
	var s strings.Builder

	s.WriteString("Hash: ")
	s.WriteString(m.Hash)
	s.WriteString("\n")

	s.WriteString("Author:\n")
	s.WriteString("	Name: ")
	s.WriteString(m.Author.Name)
	s.WriteString("\n")
	s.WriteString("	Email: ")
	s.WriteString(m.Author.Email)
	s.WriteString("\n")

	s.WriteString("Subject: ")
	s.WriteString(m.Subject)
	s.WriteString("\n")

	s.WriteString(m.Body)
	s.WriteString("\n")

	return s.String()
}

func (d FileDiff) String() string {
	var s strings.Builder

	s.WriteString("--- ")
	s.WriteString(d.OldPath)
	s.WriteString("\n")

	s.WriteString("+++ ")
	s.WriteString(d.NewPath)
	s.WriteString("\n")

	// TODO: Print hunks
	return s.String()
}

func ParsePatch(raw string) (Patch, error) {
	patch := Patch{}
	lines := strings.Split(raw, "\n")
	meta, next, err := parsePatchMetadata(lines)
	patch.Metadata = meta

	// Yes, this is intentional.  I want the partially-parsed metadata to be assigned to the patch before we return
	if err != nil {
		return patch, err
	}

	files, err := parsePatchDiff(lines[next:])
	patch.Files = files

	return patch, err
}

func parsePatchMetadata(lines []string) (CommitMetadata, int, error) {
	meta := CommitMetadata{}

	headerSeparatorIndex := -1

	for i, line := range lines {
		if len(line) == 0 {
			headerSeparatorIndex = i
			break
		}
	}

	metaSeparatorIndex := -1

	for i, line := range lines {
		if line == MetadataSeparator {
			metaSeparatorIndex = i
			break
		}
	}

	if headerSeparatorIndex == -1 || metaSeparatorIndex == -1 {
		return meta, 0, errors.New("Malformed patch header")
	}

	err := parseHeaders(lines[:headerSeparatorIndex], &meta)
	if err != nil {
		return meta, metaSeparatorIndex + 1, err
	}

	meta.Body = strings.Join(lines[headerSeparatorIndex+1:metaSeparatorIndex], "\n")

	return meta, metaSeparatorIndex + 1, nil
}

func parseHeaders(lines []string, meta *CommitMetadata) error {
	unfolded := unfoldHeaders(lines)
	for _, line := range unfolded {
		if strings.HasPrefix(line, "From ") {
			tokens := strings.Fields(line)
			if len(tokens) < 2 {
				return errors.New("Malformed hash line")
			}
			meta.Hash = tokens[1]
		} else if strings.HasPrefix(line, "From:") {
			user, err := parsePatchAuthor(strings.TrimSpace(line[len("From:"):]))
			meta.Author = user
			if err != nil {
				return err
			}

		} else if strings.HasPrefix(line, "Date:") {
			meta.Date = parsePatchDate(strings.TrimSpace(line[len("Date:"):]))
		} else if strings.HasPrefix(line, "Subject:") {
			meta.Subject = strings.TrimSpace(line[len("Subject:"):])
		}
	}

	return nil
}

func parsePatchAuthor(value string) (User, error) {
	value = strings.TrimSpace(value)

	addr, err := mail.ParseAddress(value)
	if err == nil {
		return User{
			Name:  addr.Name,
			Email: addr.Address,
		}, nil
	}

	start := strings.LastIndex(value, "<")
	end := strings.LastIndex(value, ">")
	if start == -1 || end == -1 || end < start {
		return User{Name: value}, err
	}

	name := strings.TrimSpace(value[:start])
	email := strings.TrimSpace(value[start+1 : end])

	return User{
		Name:  name,
		Email: email,
	}, nil
}

func unfoldHeaders(lines []string) []string {
	var unfolded []string

	for _, line := range lines {
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if len(unfolded) == 0 {
				unfolded = append(unfolded, strings.TrimSpace(line))
				continue
			}

			unfolded[len(unfolded)-1] += " " + strings.TrimSpace(line)
			continue
		}

		unfolded = append(unfolded, line)
	}

	return unfolded
}

func parsePatchDate(dateStr string) time.Time {
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon Jan 2 15:04:05 2006 -0700",
		"Mon Jan 2 15:04:05 2006",
		time.RFC3339,
	}

	for _, layout := range layouts {
		time, err := time.Parse(layout, dateStr)
		if err == nil {
			return time
		}
	}

	return time.Time{}
}

func parsePatchDiff(lines []string) ([]FileDiff, error) {
	var files []FileDiff

	var activeHunk *Hunk = nil

	for _, line := range lines {
		if activeHunk == nil {
			if strings.HasPrefix(line, "diff --git ") {
				files = append(files, FileDiff{})
				oldPath, newPath, err := parseDiffLine(line)
				if err != nil {
					return files, err
				}
				files[len(files)-1].OldPath = oldPath
				files[len(files)-1].NewPath = newPath
			} else if strings.HasPrefix(line, "--- ") {
				if len(files) == 0 {
					return files, errors.New("Malformed diff")
				}

				files[len(files)-1].OldPath = cleanDiffPath(line[4:])
			} else if strings.HasPrefix(line, "+++ ") {
				if len(files) == 0 {
					return files, errors.New("Malformed diff")
				}

				files[len(files)-1].NewPath = cleanDiffPath(line[4:])
			} else if strings.HasPrefix(line, "@@ ") && strings.HasSuffix(line, " @@") {
				// TODO: Parse hunk header
			}
		} else {
			// TODO: Parse Hunk lines
		}
	}

	return files, nil
}

func parseDiffLine(line string) (string, string, error) {
	stripped := strings.TrimSpace(line[len("diff --git "):])

	strs := make([]string, 0, 2)
	var s strings.Builder
	quoteCount := 0

	for _, currentRune := range stripped {
		if currentRune == ' ' {
			if quoteCount%2 == 0 {
				if s.Len() > 0 {
					strs = append(strs, s.String())
					s.Reset()
				}
				continue
			}
		} else if currentRune == '"' {
			quoteCount++
		}
		s.WriteRune(currentRune)
	}

	if s.Len() > 0 {
		strs = append(strs, s.String())
	}

	if len(strs) != 2 {
		return "", "", errors.New("Failed to parse diff line")
	}

	return cleanDiffPath(strs[0]), cleanDiffPath(strs[1]), nil
}

func cleanDiffPath(path string) string {
	path = strings.TrimSpace(path)

	if path == "/dev/null" {
		return ""
	}

	if strings.HasPrefix(path, `"`) && strings.HasSuffix(path, `"`) {
		if unquoted, err := strconv.Unquote(path); err == nil {
			path = unquoted
		}
	}

	path = strings.TrimPrefix(path, "a/")
	path = strings.TrimPrefix(path, "b/")

	return path
}
