// /home/krylon/go/src/github.com/blicero/blockbuster/objects/link.go
// -*- mode: go; coding: utf-8; -*-
// Created on 14. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-14 20:00:24 krylon>

package objects

import "net/url"

// Link represents a URL, along with a human-readable title and an optional
// description, that can be attached to a Person or a File.
type Link struct {
	ID          int64
	URL         *url.URL
	Title       string
	Description string
}

// DisplayTitle returns the Link's Title if it is non-empty,
// otherwise the URL itself.
func (l *Link) DisplayTitle() string {
	if l.Title == "" {
		return l.URL.String()
	}

	return l.Title
} // func (l *Link) DisplayTitle() string
