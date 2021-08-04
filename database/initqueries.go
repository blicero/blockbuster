// /home/krylon/go/src/github.com/blicero/blockbuster/database/initqueries.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-04 12:10:08 krylon>

package database

var initQueries = []string{
	`
CREATE TABLE file (
    id INTEGER PRIMARY KEY,
    path TEXT UNIQUE NOT NULL,
    title TEXT,
    year INTEGER,
    CHECK (year IS NULL OR year > 1900)
)`,

	"CREATE INDEX file_path_idx ON file (path)",
	"CREATE INDEX file_title_idx ON file (title)",

	`
CREATE TABLE person (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    year_born INTEGER NOT NULL,
    UNIQUE (name)
)`,

	"CREATE INDEX person_name_idx ON person (name)",

	`
CREATE TABLE person_url (
    id INTEGER PRIMARY KEY,
    person_id INTEGER NOT NULL,
    url TEXT NOT NULL,
    description TEXT,
    FOREIGN KEY (person_id) REFERENCES person (id)
)`,

	"CREATE INDEX person_url_person_idx ON person_url (person_id)",

	`
CREATE TABLE file_url (
    id INTEGER PRIMARY KEY,
    file_id INTEGER NOT NULL,
    url TEXT NOT NULL,
    description TEXT,
    FOREIGN KEY (file_id) REFERENCES file (id)
)`,

	`
CREATE TABLE tag (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
)`,

	"CREATE INDEX tag_name_idx ON tag (name)",

	`CREATE TABLE file_tag_link (
    id INTEGER PRIMARY KEY,
    file_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    UNIQUE (file_id, tag_id),
    FOREIGN KEY (file_id) REFERENCES file (id),
    FOREIGN KEY (tag_id) REFERENCES tag (id)
)`,

	"CREATE INDEX file_tag_link_file_idx ON file_tag_link (file_id)",
	"CREATE INDEX file_tag_link_tag_idx ON file_tag_link (tag_id)",
}
