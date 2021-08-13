// /home/krylon/go/src/github.com/blicero/blockbuster/objects/file.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-13 23:23:31 krylon>

package objects

import (
	"path"

	"github.com/blicero/krylib"
)

// File represents a simple video file.
type File struct {
	ID       int64
	FolderID int64
	Path     string
	Title    string
	Year     int64
	Hidden   bool
}

// DisplayTitle returns the File's Title, or its basename,
// if the Title is not set.
func (f *File) DisplayTitle() string {
	if f.Title != "" {
		return f.Title
	}

	return path.Base(f.Path)
} // func (f *File) DisplayTitle() string

func (f *File) Size() int64 {
	if size, err := krylib.FileSize(f.Path); err != nil {
		return 0
	} else {
		return size
	}
} // func (f *File) Size() int64
