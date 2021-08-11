// /home/krylon/go/src/github.com/blicero/blockbuster/database/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-10 18:30:12 krylon>

//go:generate stringer -type=ID

// Package query provides symbolic constants for the various queries we are
// going to run on the database.
package query

// ID represents a specific database query.
type ID uint8

const (
	FileAdd ID = iota
	FileRemove
	FileRemoveByFolder
	FileGetAll
	FileGetByPath
	FileGetByID
	FolderAdd
	FolderUpdateScan
	FolderRemove
	FolderGetAll
	FolderGetByPath
	TagAdd
	TagDelete
	TagGetAll
	TagGetByID
	TagGetByName
	TagLinkAdd
	TagLinkDelete
	TagLinkGetByTag
	TagLinkGetByFile
	PersonAdd
	PersonDelete
	PersonGetAll
	PersonGetByID
	PersonGetByName
	ActorAdd
	ActorDelete
	ActorGetByPerson
	ActorGetByFile
)
