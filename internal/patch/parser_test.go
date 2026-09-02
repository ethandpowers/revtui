package patch

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

const placeholderDate = "Mon Sep 17 00:00:00 2001"

func TestParseHashSha1(t *testing.T) {
	hash := "0123456789abcdef0123456789abcdef01234567"
	line := fmt.Sprintf("From %s %s", hash, placeholderDate)

	res, err := parseHash(line)
	if err != nil {
		t.Fatal(err)
	}

	if res != hash {
		t.Fatalf("expected %q, got %q", hash, res)
	}
}

func TestParseHashSha1Upper(t *testing.T) {
	hash := "ABCDEF0123456789ABCDEF0123456789ABCDEF01"
	line := fmt.Sprintf("From %s %s", hash, placeholderDate)

	res, err := parseHash(line)
	if err != nil {
		t.Fatal(err)
	}

	if res != hash {
		t.Fatalf("expected %q, got %q", hash, res)
	}
}

func TestParseHashSha256(t *testing.T) {
	hash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	line := fmt.Sprintf("From %s %s", hash, placeholderDate)

	res, err := parseHash(line)
	if err != nil {
		t.Fatal(err)
	}

	if res != hash {
		t.Fatalf("expected %q, got %q", hash, res)
	}
}

func TestParseHashAbbrevSha1(t *testing.T) {
	hash := "0123456"
	line := fmt.Sprintf("From %s %s", hash, placeholderDate)

	res, err := parseHash(line)
	if err != nil {
		t.Fatal(err)
	}

	if res != hash {
		t.Fatalf("expected %q, got %q", hash, res)
	}
}

func TestParseHashAllZeros(t *testing.T) {
	hash := "0000000000000000000000000000000000000000"
	line := fmt.Sprintf("From %s %s", hash, placeholderDate)

	res, err := parseHash(line)
	if err != nil {
		t.Fatal(err)
	}

	if res != hash {
		t.Fatalf("expected %q, got %q", hash, res)
	}
}

func TestParseHashExtraSpaces(t *testing.T) {
	hash := "0123456789abcdef0123456789abcdef01234567"
	line := fmt.Sprintf("From            %s          %s       ", hash, placeholderDate)

	res, err := parseHash(line)
	if err != nil {
		t.Fatal(err)
	}

	if res != hash {
		t.Fatalf("expected %q, got %q", hash, res)
	}
}

func TestParseHashMalformed(t *testing.T) {
	_, err := parseHash("From")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParsePatchAuthorNameAndEmail(t *testing.T) {
	name, email, err := parsePatchAuthor("Jane Doe <jane@example.com>")
	if err != nil {
		t.Fatal(err)
	}

	assertAuthor(t, name, email, "Jane Doe", "jane@example.com")
}

func TestParsePatchAuthorEmailOnly(t *testing.T) {
	name, email, err := parsePatchAuthor("jane@example.com")
	if err != nil {
		t.Fatal(err)
	}

	assertAuthor(t, name, email, "", "jane@example.com")
}

func TestParsePatchAuthorQuotedEmailName(t *testing.T) {
	name, email, err := parsePatchAuthor("\"jane@example.com\" <jane@example.com>")
	if err != nil {
		t.Fatal(err)
	}

	assertAuthor(t, name, email, "jane@example.com", "jane@example.com")
}

func TestParsePatchAuthorUnquotedEmailNameFallback(t *testing.T) {
	name, email, err := parsePatchAuthor("jane@example.com <jane@example.com>")
	if err != nil {
		t.Fatal(err)
	}

	assertAuthor(t, name, email, "jane@example.com", "jane@example.com")
}

func TestParsePatchAuthorExtraSpaces(t *testing.T) {
	name, email, err := parsePatchAuthor("   Jane Doe    <jane@example.com>   ")
	if err != nil {
		t.Fatal(err)
	}

	assertAuthor(t, name, email, "Jane Doe", "jane@example.com")
}

func TestParsePatchAuthorFullHeaderLine(t *testing.T) {
	name, email, err := parsePatchAuthor("From: Jane Doe <jane@example.com>")
	if err != nil {
		t.Fatal(err)
	}

	assertAuthor(t, name, email, "Jane Doe", "jane@example.com")
}

func TestParsePatchAuthorInvalidFallback(t *testing.T) {
	value := "not a valid address"
	name, email, err := parsePatchAuthor(value)
	if err == nil {
		t.Fatal("expected error")
	}

	assertAuthor(t, name, email, value, "")
}

func TestUnfoldHeadersContinuationLine(t *testing.T) {
	lines := []string{
		"Subject: [PATCH] Add parser",
		" with folded subject",
		"Date: Tue, 20 Aug 2026 12:30:00 -0700",
	}

	got := unfoldHeaders(lines)
	want := []string{
		"Subject: [PATCH] Add parser with folded subject",
		"Date: Tue, 20 Aug 2026 12:30:00 -0700",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestUnfoldHeadersLeadingContinuation(t *testing.T) {
	got := unfoldHeaders([]string{" orphan", "Subject: Test"})
	want := []string{"orphan", "Subject: Test"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestParsePatchDateRFC1123Z(t *testing.T) {
	date := parsePatchDate("Tue, 20 Aug 2026 12:30:00 -0700")
	if date.IsZero() {
		t.Fatal("expected parsed date")
	}

	if date.Year() != 2026 || date.Month() != time.August || date.Day() != 20 {
		t.Fatalf("unexpected date: %s", date)
	}
}

func TestParsePatchDateRFC3339(t *testing.T) {
	date := parsePatchDate("2026-08-20T12:30:00-07:00")
	if date.IsZero() {
		t.Fatal("expected parsed date")
	}

	if date.Year() != 2026 || date.Month() != time.August || date.Day() != 20 {
		t.Fatalf("unexpected date: %s", date)
	}
}

func TestParsePatchDateInvalid(t *testing.T) {
	date := parsePatchDate("not a date")
	if !date.IsZero() {
		t.Fatalf("expected zero date, got %s", date)
	}
}

func TestParseHeaders(t *testing.T) {
	var meta CommitMetadata
	err := parseHeaders([]string{
		"From 0123456789abcdef0123456789abcdef01234567 Mon Sep 17 00:00:00 2001",
		"From: Jane Doe <jane@example.com>",
		"Date: Tue, 20 Aug 2026 12:30:00 -0700",
		"Subject: [PATCH] Add parser",
	}, &meta)
	if err != nil {
		t.Fatal(err)
	}

	if meta.Hash != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("unexpected hash: %q", meta.Hash)
	}
	assertAuthor(t, meta.AuthorName, meta.AuthorEmail, "Jane Doe", "jane@example.com")
	if meta.Subject != "[PATCH] Add parser" {
		t.Fatalf("unexpected subject: %q", meta.Subject)
	}
	if meta.Date.IsZero() {
		t.Fatal("expected date")
	}
}

func TestParsePatchMetadataWithBody(t *testing.T) {
	lines := stringsToLines(testPatchRaw())
	meta, next, err := parsePatchMetadata(lines)
	if err != nil {
		t.Fatal(err)
	}

	if next != 9 {
		t.Fatalf("expected next index 9, got %d", next)
	}
	if meta.Body != "Body line one.\n\nBody line two." {
		t.Fatalf("unexpected body: %q", meta.Body)
	}
}

func TestParsePatchMetadataWithoutBody(t *testing.T) {
	lines := stringsToLines(strings.Join([]string{
		"From 0123456789abcdef0123456789abcdef01234567 Mon Sep 17 00:00:00 2001",
		"From: Jane Doe <jane@example.com>",
		"Date: Tue, 20 Aug 2026 12:30:00 -0700",
		"Subject: [PATCH] Add parser",
		"",
		"---",
		"diff --git a/main.go b/main.go",
	}, "\n"))

	meta, next, err := parsePatchMetadata(lines)
	if err != nil {
		t.Fatal(err)
	}

	if next != 6 {
		t.Fatalf("expected next index 6, got %d", next)
	}
	if meta.Body != "" {
		t.Fatalf("expected empty body, got %q", meta.Body)
	}
}

func TestParsePatchMetadataMissingHeaderSeparator(t *testing.T) {
	_, _, err := parsePatchMetadata([]string{
		"From 0123456789abcdef0123456789abcdef01234567 Mon Sep 17 00:00:00 2001",
		"From: Jane Doe <jane@example.com>",
		"---",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParsePatchMetadataMissingMetadataSeparator(t *testing.T) {
	_, _, err := parsePatchMetadata([]string{
		"From 0123456789abcdef0123456789abcdef01234567 Mon Sep 17 00:00:00 2001",
		"From: Jane Doe <jane@example.com>",
		"",
		"body",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseHunkRangeWithCount(t *testing.T) {
	start, count, err := parseHunkRange("-10,7")
	if err != nil {
		t.Fatal(err)
	}

	if start != 10 || count != 7 {
		t.Fatalf("expected 10,7 got %d,%d", start, count)
	}
}

func TestParseHunkRangeWithoutCount(t *testing.T) {
	start, count, err := parseHunkRange("+10")
	if err != nil {
		t.Fatal(err)
	}

	if start != 10 || count != 1 {
		t.Fatalf("expected 10,1 got %d,%d", start, count)
	}
}

func TestParseHunkRangeZeroCount(t *testing.T) {
	start, count, err := parseHunkRange("-0,0")
	if err != nil {
		t.Fatal(err)
	}

	if start != 0 || count != 0 {
		t.Fatalf("expected 0,0 got %d,%d", start, count)
	}
}

func TestParseHunkHeaderWithoutContext(t *testing.T) {
	hunk, err := parseHunkHeader("@@ -10,7 +10,8 @@")
	if err != nil {
		t.Fatal(err)
	}

	assertHunkRange(t, hunk, 10, 7, 10, 8)
}

func TestParseHunkHeaderWithContext(t *testing.T) {
	hunk, err := parseHunkHeader("@@ -42 +42,2 @@ func main() {")
	if err != nil {
		t.Fatal(err)
	}

	assertHunkRange(t, hunk, 42, 1, 42, 2)
}

func TestParseDiffLineSimplePaths(t *testing.T) {
	oldPath, newPath, err := parseDiffLine("diff --git a/old.go b/new.go")
	if err != nil {
		t.Fatal(err)
	}

	if oldPath != "old.go" || newPath != "new.go" {
		t.Fatalf("expected old.go,new.go got %q,%q", oldPath, newPath)
	}
}

func TestParseDiffLineQuotedPaths(t *testing.T) {
	oldPath, newPath, err := parseDiffLine(`diff --git "a/old file.go" "b/new file.go"`)
	if err != nil {
		t.Fatal(err)
	}

	if oldPath != "old file.go" || newPath != "new file.go" {
		t.Fatalf("expected quoted paths to be unquoted, got %q,%q", oldPath, newPath)
	}
}

func TestParseDiffLineMalformed(t *testing.T) {
	_, _, err := parseDiffLine("diff --git a/only-one.go")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCleanDiffPathDevNull(t *testing.T) {
	if got := cleanDiffPath("/dev/null"); got != "" {
		t.Fatalf("expected empty path, got %q", got)
	}
}

func TestCleanDiffPathPrefixes(t *testing.T) {
	if got := cleanDiffPath("a/path/to/file.go"); got != "path/to/file.go" {
		t.Fatalf("unexpected old path: %q", got)
	}

	if got := cleanDiffPath("b/path/to/file.go"); got != "path/to/file.go" {
		t.Fatalf("unexpected new path: %q", got)
	}
}

func TestCleanDiffPathQuotedUnicodeEscape(t *testing.T) {
	if got := cleanDiffPath(`"a/caf\303\251.go"`); got != "café.go" {
		t.Fatalf("unexpected path: %q", got)
	}
}

func TestParsePatchDiffSingleFileHunk(t *testing.T) {
	files, err := parsePatchDiff([]string{
		"diff --git a/main.go b/main.go",
		"index 1111111..2222222 100644",
		"--- a/main.go",
		"+++ b/main.go",
		"@@ -1,2 +1,3 @@ func main() {",
		" package main",
		"-old",
		"+new",
		"+extra",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	file := files[0]
	if file.OldPath != "main.go" || file.NewPath != "main.go" {
		t.Fatalf("unexpected paths: %q,%q", file.OldPath, file.NewPath)
	}
	if len(file.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(file.Hunks))
	}

	hunk := file.Hunks[0]
	assertHunkRange(t, hunk, 1, 2, 1, 3)
	assertDiffLines(t, hunk.Lines, []DiffLine{
		{Type: DiffLineContext, Text: "package main"},
		{Type: DiffLineRemoved, Text: "old"},
		{Type: DiffLineAdded, Text: "new"},
		{Type: DiffLineAdded, Text: "extra"},
	})
}

func TestParsePatchDiffNoNewlineMarkerOnRemovedLine(t *testing.T) {
	files, err := parsePatchDiff([]string{
		"diff --git a/main.go b/main.go",
		"--- a/main.go",
		"+++ b/main.go",
		"@@ -1 +1 @@",
		"-old",
		`\ No newline at end of file`,
		"+new",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 || len(files[0].Hunks) != 1 {
		t.Fatalf("expected one parsed hunk, got %#v", files)
	}

	assertDiffLines(t, files[0].Hunks[0].Lines, []DiffLine{
		{Type: DiffLineRemoved, Text: "old", NoNewlineAtEOF: true},
		{Type: DiffLineAdded, Text: "new"},
	})
}

func TestParsePatchDiffNoNewlineMarkerOnAddedLine(t *testing.T) {
	files, err := parsePatchDiff([]string{
		"diff --git a/main.go b/main.go",
		"--- a/main.go",
		"+++ b/main.go",
		"@@ -1 +1 @@",
		"-old",
		"+new",
		`\ No newline at end of file`,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 || len(files[0].Hunks) != 1 {
		t.Fatalf("expected one parsed hunk, got %#v", files)
	}

	assertDiffLines(t, files[0].Hunks[0].Lines, []DiffLine{
		{Type: DiffLineRemoved, Text: "old"},
		{Type: DiffLineAdded, Text: "new", NoNewlineAtEOF: true},
	})
}

func TestParsePatchDiffMultipleFiles(t *testing.T) {
	files, err := parsePatchDiff([]string{
		"diff --git a/one.go b/one.go",
		"--- a/one.go",
		"+++ b/one.go",
		"diff --git a/two.go b/two.go",
		"--- a/two.go",
		"+++ b/two.go",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].OldPath != "one.go" || files[1].OldPath != "two.go" {
		t.Fatalf("unexpected files: %#v", files)
	}
}

func TestParsePatchFullPatch(t *testing.T) {
	patch, err := ParsePatch(testPatchRaw())
	if err != nil {
		t.Fatal(err)
	}

	if patch.Metadata.Hash != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("unexpected hash: %q", patch.Metadata.Hash)
	}
	assertAuthor(t, patch.Metadata.AuthorName, patch.Metadata.AuthorEmail, "Jane Doe", "jane@example.com")
	if patch.Metadata.Body != "Body line one.\n\nBody line two." {
		t.Fatalf("unexpected body: %q", patch.Metadata.Body)
	}
	if len(patch.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(patch.Files))
	}
	if len(patch.Files[0].Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(patch.Files[0].Hunks))
	}
}

func assertAuthor(t *testing.T, name string, email string, wantName string, wantEmail string) {
	t.Helper()

	if name != wantName {
		t.Fatalf("expected name %q, got %q", wantName, name)
	}

	if email != wantEmail {
		t.Fatalf("expected email %q, got %q", wantEmail, email)
	}
}

func assertHunkRange(t *testing.T, hunk Hunk, oldStart int, oldCount int, newStart int, newCount int) {
	t.Helper()

	if hunk.OldStart != oldStart || hunk.OldCount != oldCount || hunk.NewStart != newStart || hunk.NewCount != newCount {
		t.Fatalf(
			"expected hunk -%d,%d +%d,%d got -%d,%d +%d,%d",
			oldStart,
			oldCount,
			newStart,
			newCount,
			hunk.OldStart,
			hunk.OldCount,
			hunk.NewStart,
			hunk.NewCount,
		)
	}
}

func assertDiffLines(t *testing.T, got []DiffLine, want []DiffLine) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func stringsToLines(value string) []string {
	return strings.Split(value, "\n")
}

func testPatchRaw() string {
	return strings.Join([]string{
		"From 0123456789abcdef0123456789abcdef01234567 Mon Sep 17 00:00:00 2001",
		"From: Jane Doe <jane@example.com>",
		"Date: Tue, 20 Aug 2026 12:30:00 -0700",
		"Subject: [PATCH] Add parser",
		"",
		"Body line one.",
		"",
		"Body line two.",
		"---",
		" cmd/main.go | 3 ++-",
		" 1 file changed, 2 insertions(+), 1 deletion(-)",
		"",
		"diff --git a/cmd/main.go b/cmd/main.go",
		"index 1111111..2222222 100644",
		"--- a/cmd/main.go",
		"+++ b/cmd/main.go",
		"@@ -1,2 +1,3 @@ func main() {",
		" package main",
		"-old",
		"+new",
		"+extra",
	}, "\n")
}
