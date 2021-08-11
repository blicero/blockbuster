// /home/krylon/go/src/github.com/blicero/blockbuster/database/dbqueries.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-10 00:25:58 krylon>

package database

import "github.com/blicero/blockbuster/database/query"

var dbQueries = map[query.ID]string{
	query.FileAdd: `
INSERT INTO file (path, folder_id)
VALUES           (   ?,         ?)
`,
	query.FileRemove:         "DELETE FROM file WHERE id = ?",
	query.FileRemoveByFolder: "DELETE FROM file WHERE folder_id = ?",
	query.FileGetAll:         "SELECT id, folder_id, path, title, year, hidden FROM file",
	query.FileGetByPath:      "SELECT id, folder_id, title, year, hidden FROM file WHERE path = ?",
	query.FileGetByID:        "SELECT folder_id, path, title, year, hidden FROM file WHERE id = ?",
	query.FolderAdd:          "INSERT INTO folder(path) VALUES (?)",
	query.FolderRemove:       "DELETE FROM folder WHERE id = ?",
	query.FolderUpdateScan:   "UPDATE folder SET last_scan = ? WHERE id = ?",
	query.FolderGetAll:       "SELECT id, path, last_scan FROM folder",
	query.FolderGetByPath:    "SELECT id, last_scan FROM folder WHERE path = ?",
	query.TagAdd:             "INSERT INTO tag (name) VALUES (?)",
	query.TagDelete:          "DELETE FROM tag WHERE id = ?",
	query.TagGetAll:          "SELECT id, name FROM tag",
	query.TagGetByID:         "SELECT name FROM tag WHERE id = ?",
	query.TagGetByName:       "SELECT id FROM tag WHERE name = ?",
	query.TagLinkAdd:         "INSERT INTO tag_link (file_id, tag_id) VALUES (?, ?)",
	query.TagLinkDelete:      "DELETE FROM tag_link WHERE id = ?",
	query.TagLinkGetByTag:    "SELECT id, file_id FROM tag_link WHERE tag_id = ?",
	query.TagLinkGetByFile:   "SELECT id, tag_id FROM tag_link WHERE file_id = ?",
}
