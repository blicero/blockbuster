// /home/krylon/go/src/github.com/blicero/blockbuster/objects/file.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-07 21:35:08 krylon>

package objects

// File represents a simple video file.
type File struct {
	ID       int64
	FolderID int64
	Path     string
	Title    string
	Year     int64
}
