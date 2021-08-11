// /home/krylon/go/src/github.com/blicero/blockbuster/objects/tag.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-11 09:29:09 krylon>

package objects

// Tag is a ... tag that can be attached to videos. Duh.
type Tag struct {
	ID   int64
	Name string
}

// TagList is a helper type to sort Tags by Name.
type TagList []Tag

func (tl TagList) Len() int           { return len(tl) }
func (tl TagList) Less(i, j int) bool { return tl[i].Name < tl[j].Name }
func (tl TagList) Swap(i, j int)      { tl[i], tl[j] = tl[j], tl[i] }
