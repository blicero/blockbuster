// /home/krylon/go/src/github.com/blicero/blockbuster/database/initqueries.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-03 00:09:30 krylon>

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

	`
CREATE TABLE person (
    id INTEGER PRIMARY KEY
    name TEXT UNIQUE NOT NULL,
    year_born INTEGER NOT NULL
)`,

	`
CREATE TABLE person_url (
    id INTEGER PRIMARY KEY,
    person_id INTEGER NOT NULL,
    url TEXT NOT NULL,
    description TEXT,
    FOREIGN KEY (person_id) REFERENCES person (id)
)`,

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
}
