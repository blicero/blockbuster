// /home/krylon/go/src/github.com/blicero/blockbuster/database/02_folder_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 07. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-07 19:15:06 krylon>

package database

import (
	"testing"

	"github.com/blicero/blockbuster/objects"
)

const basePath = "/test/Video"

var folder *objects.Folder

func TestFolderAdd(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var err error

	if folder, err = tdb.FolderAdd(basePath); err != nil {
		folder = nil
		t.Fatalf("Cannot add Folder %s to Database: %s",
			basePath,
			err.Error())
	}
} // func TestFolderAdd(t *testing.T)
