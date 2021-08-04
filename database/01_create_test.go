// /home/krylon/go/src/github.com/blicero/blockbuster/database/01_create_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-04 12:01:28 krylon>

package database

import (
	"testing"

	"github.com/blicero/blockbuster/common"
	"github.com/blicero/blockbuster/database/query"
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
		err    error
		idList = []query.ID{
			query.FileAdd,
			query.FileRemove,
			query.FileGetAll,
			query.FileGetByPath,
			query.FileGetByID,
		}
	)

	if tdb == nil {
		t.SkipNow()
	}

	for _, qid := range idList {
		if _, err = tdb.getQuery(qid); err != nil {
			t.Errorf("Cannot prepare query %s: %s",
				qid,
				err.Error())
		}
	}
} // func TestQueryPrepare(t *testing.T)
