// /home/krylon/go/src/github.com/blicero/blockbuster/database/dbqueries.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-04 11:38:13 krylon>

package database

import "github.com/blicero/blockbuster/database/query"

var dbQueries = map[query.ID]string{
	query.FileAdd: `
INSERT INTO file (path)
VALUES           (?)
`,
	query.FileRemove:    "DELETE FROM file WHERE id = ?",
	query.FileGetAll:    "SELECT id, path, title, year FROM file",
	query.FileGetByPath: "SELECT id, title, year FROM file WHERE path = ?",
	query.FileGetByID:   "SELECT path, title, year FROM file WHERE id = ?",
}
