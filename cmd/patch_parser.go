package main

import "time"

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

func parse_patch() Patch {
	return Patch{}
}
