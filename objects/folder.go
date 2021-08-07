// /home/krylon/go/src/github.com/blicero/blockbuster/objects/folder.go
// -*- mode: go; coding: utf-8; -*-
// Created on 07. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-07 21:31:12 krylon>

package objects

import "time"

// Folder represents the root of a directory tree that is scanned
// for Files.
type Folder struct {
	ID       int64
	Path     string
	LastScan time.Time
}

// IsKnown returns true if the Folder's timestamp from the most recent scan
// is not the epoch, i.e. if the Folder has been scanned before.
func (f *Folder) IsKnown() bool {
	return f.LastScan.Unix() != 0
}
