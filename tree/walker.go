// /home/krylon/go/src/github.com/blicero/blockbuster/tree/walker.go
// -*- mode: go; coding: utf-8; -*-
// Created on 07. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-07 20:22:29 krylon>

package tree

import (
	"io/fs"
	"log"

	"github.com/blicero/blockbuster/database"
	"github.com/blicero/blockbuster/objects"
	"github.com/blicero/krylib"
)

const (
	minSize       = 1024 * 1024 * 32 // 32 MB, minimum size for files to consider
	suffixPattern = "(?i)[.](?:avi|mp4|mpg|asf|avi|flv|m4v|mkv|mov|mpg|ogm|ogv|sfv|webm|wmv)$"
)

type walker struct {
	log   *log.Logger
	root  *objects.Folder
	fileQ chan<- *objects.File
	db    *database.Database
}

func (w *walker) visitFile(path string, d fs.DirEntry, incoming error) error {
	if incoming != nil {
		w.log.Printf("[ERROR] Incoming error when visiting %s: %s\n",
			path,
			incoming.Error())
		return fs.SkipDir
	} else if !suffixRe.MatchString(path) {
		w.log.Printf("[TRACE] Skip %q -- suffix\n", path)
		return nil
	} else if !d.Type().IsRegular() {
		w.log.Printf("[TRACE] Skip %q -- not a regular file.\n", path)
		return nil
	}

	var (
		err  error
		file *objects.File
		info fs.FileInfo
	)

	if info, err = d.Info(); err != nil {
		w.log.Printf("[ERROR] Cannot read Info for %s: %s\n",
			path,
			err.Error())
		return err
	} else if info.Size() < minSize {
		w.log.Printf("[TRACE] Skip %q -- too small (%s)\n",
			path,
			krylib.FmtBytes(info.Size()))
		return nil
	} else if file, err = w.db.FileAdd(path, w.root); err != nil {
		w.log.Printf("[ERROR] Cannot add File %q to Database: %s\n",
			path,
			err.Error())
		return err
	}

	w.fileQ <- file

	return nil
} // func (w *walker) visitFile(path string, d fs.DirEntry, incoming error) error
