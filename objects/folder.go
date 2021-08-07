// /home/krylon/go/src/github.com/blicero/blockbuster/objects/folder.go
// -*- mode: go; coding: utf-8; -*-
// Created on 07. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-07 18:00:26 krylon>

package objects

import "time"

// Folder represents the root of a directory tree that is scanned
// for Files.
type Folder struct {
	ID       int64
	Path     string
	LastScan time.Time
}
