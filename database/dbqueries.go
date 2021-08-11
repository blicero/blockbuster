// /home/krylon/go/src/github.com/blicero/blockbuster/database/dbqueries.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-11 18:13:55 krylon>

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
	query.TagLinkDelete:      "DELETE FROM tag_link WHERE file_id = ? AND tag_id = ?",
	query.TagLinkGetByTag: `
SELECT
    f.id,
    f.folder_id,
    f.path,
    f.title,
    f.year
FROM file_tag_link l
INNER JOIN file f ON l.file_id = f.id
WHERE l.tag_id = ?
`,
	query.TagLinkGetByFile: `
SELECT
    t.id,
    t.name
FROM tag_link l
INNER JOIN tag t ON l.tag_id = t.id
WHERE l.file_id = ?
`,
	query.PersonAdd:        "INSERT INTO person (name, birthday) VALUES (?, ?)",
	query.PersonDelete:     "DELETE FROM person WHERE id = ?",
	query.PersonGetAll:     "SELECT id, name, birthday FROM person",
	query.PersonGetByID:    "SELECT name, birthday FROM person WHERE id = ?",
	query.PersonGetByName:  "SELECT id, birthday FROM person WHERE name = ?",
	query.ActorAdd:         "INSERT INTO actor (file_id, person_id) VALUE (?, ?)",
	query.ActorDelete:      "DELETE FROM actor WHERE id = ?",
	query.ActorGetByPerson: "SELECT id, file_id FROM actor WHERE person_id = ?",
	query.ActorGetByFile:   "SELECT id, person_id FROM actor WHERE file_id = ?",
}
