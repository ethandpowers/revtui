package patch

import (
	"errors"
	"fmt"
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
	Hash        string
	AuthorName  string
	AuthorEmail string
	Date        time.Time
	Subject     string
	Body        string
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
	Type           DiffLineType
	Text           string
	NoNewlineAtEOF bool
}

type DiffLineType int

const (
	DiffLineContext DiffLineType = iota
	DiffLineAdded
	DiffLineRemoved
)

func (d DiffLineType) String() string {
	switch d {
	case DiffLineContext:
		return " "
	case DiffLineAdded:
		return "+"
	case DiffLineRemoved:
		return "-"
	}

	return "unknown"
}

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
	s.WriteString(m.AuthorName)
	s.WriteString("\n")
	s.WriteString("	Email: ")
	s.WriteString(m.AuthorEmail)
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

	for _, hunk := range d.Hunks {
		s.WriteString(hunk.String())
		s.WriteString("\n")
	}
	return s.String()
}

func (h Hunk) String() string {
	var s strings.Builder

	s.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", h.OldStart, h.OldCount, h.NewStart, h.NewCount))
	for _, line := range h.Lines {
		s.WriteString(line.Type.String())
		s.WriteString(line.Text)
		s.WriteString("\n")
	}

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

	if headerSeparatorIndex == -1 {
		return meta, 0, errors.New("Malformed patch header")
	}

	metaSeparatorIndex := -1

	for i := headerSeparatorIndex + 1; i < len(lines); i++ {
		line := lines[i]
		if line == MetadataSeparator {
			metaSeparatorIndex = i
			break
		}
	}

	if metaSeparatorIndex == -1 {
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
			hash, err := parseHash(line)
			meta.Hash = hash
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(line, "From:") {
			name, email, err := parsePatchAuthor(line)
			meta.AuthorName = name
			meta.AuthorEmail = email
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

func parseHash(line string) (string, error) {
	tokens := strings.Fields(line)
	if len(tokens) < 2 {
		return "", errors.New("Malformed hash line")
	}

	return tokens[1], nil
}

func parsePatchAuthor(line string) (string, string, error) {
	value := strings.TrimSpace(strings.TrimPrefix(line, "From:"))

	addr, err := mail.ParseAddress(value)
	if err == nil {
		return addr.Name, addr.Address, nil
	}

	start := strings.LastIndex(value, "<")
	end := strings.LastIndex(value, ">")
	if start == -1 || end == -1 || end < start {
		return value, "", err
	}

	name := strings.TrimSpace(value[:start])
	email := strings.TrimSpace(value[start+1 : end])

	return name, email, nil
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
	oldSeen := 0
	newSeen := 0

	for _, line := range lines {
		if activeHunk == nil {
			if line == `\ No newline at end of file` {
				if len(files) > 0 && len(files[len(files)-1].Hunks) > 0 {
					hunks := files[len(files)-1].Hunks
					lines := hunks[len(hunks)-1].Lines
					if len(lines) > 0 {
						files[len(files)-1].Hunks[len(hunks)-1].Lines[len(lines)-1].NoNewlineAtEOF = true
					}
				}
			} else if strings.HasPrefix(line, "diff --git ") {
				files = append(files, FileDiff{})
				oldPath, newPath, err := parseDiffLine(line)
				if err != nil {
					return files, err
				}
				files[len(files)-1].OldPath = oldPath
				files[len(files)-1].NewPath = newPath
			} else if strings.HasPrefix(line, "--- ") {
				if len(files) == 0 {
					return files, errors.New("Malformed diff on")
				}

				files[len(files)-1].OldPath = cleanDiffPath(line[4:])
			} else if strings.HasPrefix(line, "+++ ") {
				if len(files) == 0 {
					return files, errors.New("Malformed diff")
				}

				files[len(files)-1].NewPath = cleanDiffPath(line[4:])
			} else if strings.HasPrefix(line, "@@ ") && strings.Contains(line[3:], " @@") {
				hunk, err := parseHunkHeader(line)
				if err != nil {
					return files, err
				}
				activeHunk = &hunk
				oldSeen = 0
				newSeen = 0
			}
		} else {
			if len(line) == 0 {
				continue
			}
			if line == `\ No newline at end of file` {
				if len(activeHunk.Lines) > 0 {
					activeHunk.Lines[len(activeHunk.Lines)-1].NoNewlineAtEOF = true
				}
				continue
			}
			prefix := line[0:1]
			srcLine := line[1:]

			hunkLine := DiffLine{
				Text: srcLine,
			}

			switch prefix {
			case " ":
				hunkLine.Type = DiffLineContext
				newSeen++
				oldSeen++
			case "+":
				hunkLine.Type = DiffLineAdded
				newSeen++
			case "-":
				hunkLine.Type = DiffLineRemoved
				oldSeen++
			default:
				return files, errors.New("Malformed hunk line")
			}

			activeHunk.Lines = append(activeHunk.Lines, hunkLine)

			if oldSeen == activeHunk.OldCount && newSeen == activeHunk.NewCount {
				files[len(files)-1].Hunks = append(files[len(files)-1].Hunks, *activeHunk)
				activeHunk = nil
			}
		}
	}

	return files, nil
}

func parseHunkHeader(line string) (Hunk, error) {
	hunk := Hunk{}
	end := strings.Index(line[3:], " @@")
	if end == -1 {
		return hunk, errors.New(fmt.Sprintf("Malformed hunk header: %s", line))
	}

	strippedLine := line[3 : 3+end]
	pairs := strings.Fields(strippedLine)
	if len(pairs) != 2 {
		return hunk, errors.New(fmt.Sprintf("Malformed hunk header: %s -> %s", line, strippedLine))
	}

	for i, pair := range pairs {
		start, count, err := parseHunkRange(pair)
		if err != nil {
			return hunk, errors.New(fmt.Sprintf("Malformed hunk header: %s -> %s", line, strippedLine))
		}

		if i == 0 {
			hunk.OldStart = start
			hunk.OldCount = count
		} else {
			hunk.NewStart = start
			hunk.NewCount = count
		}
	}

	return hunk, nil
}

func parseHunkRange(value string) (start int, count int, err error) {
	value = strings.TrimSpace(value)

	parts := strings.SplitN(value, ",", 2)

	start, err = strconv.Atoi(parts[0][1:])
	if err != nil {
		return 0, 0, err
	}

	count = 1
	if len(parts) == 2 {
		count, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, err
		}
	}

	return start, count, nil
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
