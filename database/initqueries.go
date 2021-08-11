// /home/krylon/go/src/github.com/blicero/blockbuster/database/initqueries.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-11 18:33:51 krylon>

package database

var initQueries = []string{
	`CREATE TABLE folder(
    id            INTEGER PRIMARY KEY,
    path          TEXT UNIQUE NOT NULL,
    last_scan     INTEGER NOT NULL DEFAULT 0
)`,

	"CREATE INDEX folder_path_idx ON folder (path)",

	`
CREATE TABLE file (
    id INTEGER PRIMARY KEY,
    folder_id INTEGER NOT NULL,
    path TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    year INTEGER NOT NULL DEFAULT 0,
    hidden INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (folder_id) REFERENCES folder (id)
       ON DELETE RESTRICT
       ON UPDATE RESTRICT
)`,

	"CREATE INDEX file_path_idx ON file (path)",
	"CREATE INDEX file_title_idx ON file (title)",
	"CREATE INDEX file_hidden_idx ON file (hidden)",

	`
CREATE TABLE person (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    birthday INTEGER NOT NULL DEFAULT 0,
    UNIQUE (name)
)`,

	"CREATE INDEX person_name_idx ON person (name)",

	`
CREATE TABLE person_url (
    id INTEGER PRIMARY KEY,
    person_id INTEGER NOT NULL,
    url TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (person_id) REFERENCES person (id)
        ON DELETE RESTRICT
        ON UPDATE RESTRICT
)`,

	"CREATE INDEX person_url_person_idx ON person_url (person_id)",

	`
CREATE TABLE actor (
    id		INTEGER PRIMARY KEY,
    file_id	INTEGER NOT NULL,
    person_id	INTEGER NOT NULL,
    UNIQUE (file_id, person_id),
    FOREIGN KEY (file_id) REFERENCES file (id)
        ON DELETE RESTRICT
        ON UPDATE RESTRICT
)
`,
	"CREATE INDEX actor_file_idx ON actor (file_id)",
	"CREATE INDEX actor_person_idx ON actor (person_id)",

	`
CREATE TABLE file_url (
    id INTEGER PRIMARY KEY,
    file_id INTEGER NOT NULL,
    url TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (file_id) REFERENCES file (id)
       ON DELETE RESTRICT
       ON UPDATE RESTRICT
)`,

	`
CREATE TABLE tag (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
)`,

	"CREATE INDEX tag_name_idx ON tag (name)",

	`CREATE TABLE tag_link (
    id INTEGER PRIMARY KEY,
    file_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    UNIQUE (file_id, tag_id),
    FOREIGN KEY (file_id) REFERENCES file (id)
        ON DELETE RESTRICT
        ON UPDATE RESTRICT,
    FOREIGN KEY (tag_id) REFERENCES tag (id)
        ON DELETE RESTRICT
        ON UPDATE RESTRICT
)`,

	"CREATE INDEX file_tag_link_file_idx ON tag_link (file_id)",
	"CREATE INDEX file_tag_link_tag_idx ON tag_link (tag_id)",
}
