// /home/krylon/go/src/github.com/blicero/blockbuster/database/02_file_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-05 10:04:35 krylon>

package database

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/blicero/blockbuster/objects"
)

const (
	fileCnt  = 100
	basePath = "/test/Video"
)

var testFiles []objects.File

func init() {
	testFiles = make([]objects.File, 0, fileCnt)
}

func TestFileAdd(t *testing.T) {
	var err error

	if tdb == nil {
		t.SkipNow()
	}

	for i := 0; i < fileCnt; i++ {
		var (
			f        *objects.File
			filename = filepath.Join(basePath, fmt.Sprintf("test_video_%03d.mp4", i))
		)

		if f, err = tdb.FileAdd(filename); err != nil {
			t.Fatalf("Error adding File %s to Database: %s",
				filename,
				err.Error())
		}

		testFiles = append(testFiles, *f)
	}
} // func TestFileAdd(t *testing.T)

func TestFileGetAll(t *testing.T) {
	var (
		fetchFiles []objects.File
		err        error
	)

	if tdb == nil {
		t.SkipNow()
	}

	if fetchFiles, err = tdb.FileGetAll(); err != nil {
		t.Fatalf("Cannot get all Files from Database: %s",
			err.Error())
	} else if len(fetchFiles) != len(testFiles) {
		t.Fatalf("FileGetAll returned an unexpected number of Files: %d (expected %d)",
			len(fetchFiles),
			len(testFiles))
	}
} // func TestFileGetAll(t *testing.T)
