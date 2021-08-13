// /home/krylon/go/src/github.com/blicero/blockbuster/database/01_create_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-13 12:14:02 krylon>

package database

import (
	"testing"

	"github.com/blicero/blockbuster/common"
)

var tdb *Database

func TestDBCreate(t *testing.T) {
	var err error

	if tdb, err = Open(common.DbPath); err != nil {
		tdb = nil
		t.Fatalf("Cannot create Database: %s",
			err.Error())
	}
} // func TestDBCreate(t *testing.T)

func TestQueryPrepare(t *testing.T) {
	var (
		err error
		// idList = []query.ID{
		// 	query.FileAdd,
		// 	query.FileRemove,
		// 	query.FileGetAll,
		// 	query.FileGetByPath,
		// 	query.FileGetByID,
		// 	query.FolderAdd,
		// 	query.FolderRemove,
		// 	query.FolderUpdateScan,
		// 	query.FolderGetAll,
		// 	query.FolderGetByPath,
		// 	query.TagAdd,
		// 	query.TagDelete,
		// 	query.TagGetAll,
		// 	query.TagGetByID,
		// 	query.TagGetByName,
		// 	query.TagLinkAdd,
		// 	query.TagLinkDelete,
		// 	query.TagLinkGetByTag,
		// 	query.TagLinkGetByFile,
		// 	query.PersonAdd,
		// 	query.PersonDelete,
		// 	query.PersonGetByID,
		// 	query.PersonGetByName,
		// 	query.ActorAdd,
		// 	query.ActorDelete,
		// 	query.ActorGetByPerson,
		// 	query.ActorGetByFile,
		// }
	)

	if tdb == nil {
		t.SkipNow()
	}

	for qid := range dbQueries {
		if _, err = tdb.getQuery(qid); err != nil {
			t.Errorf("Cannot prepare query %s: %s",
				qid,
				err.Error())
		}
	}
} // func TestQueryPrepare(t *testing.T)
