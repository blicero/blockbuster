// /home/krylon/go/src/github.com/blicero/blockbuster/logdomain/id.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-05 22:07:55 krylon>

// Package logdomain provides constants for log sources.
package logdomain

//go:generate stringer -type=ID

// ID represents a log source
type ID uint8

// These constants signify the various parts of the application.
const (
	Common ID = iota
	Database
	GUI
	Scanner
)

// AllDomains returns a slice of all the known log sources.
func AllDomains() []ID {
	return []ID{
		Common,
		Database,
		GUI,
		Scanner,
	}
} // func AllDomains() []ID
