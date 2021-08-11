// /home/krylon/go/src/github.com/blicero/blockbuster/ui/file.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-11 18:08:43 krylon>

package ui

import (
	"fmt"

	"github.com/blicero/blockbuster/objects"
	"github.com/gotk3/gotk3/gtk"
)

func (g *GUI) mkFileTagMenu(f *objects.File) (*gtk.Menu, error) {
	var (
		err  error
		msg  string
		tags map[int64]objects.Tag
		menu *gtk.Menu
	)

	if tags, err = g.db.TagLinkGetByFile(f); err != nil {
		msg = fmt.Sprintf("Cannot get list of Tags for %s: %s",
			f.DisplayTitle(),
			err.Error())
		goto ERROR
	} else if menu, err = gtk.MenuNew(); err != nil {
		msg = fmt.Sprintf("Cannot create Menu for Tags: %s",
			err.Error())
		goto ERROR
	}

	// TODO Register handlers!
	for _, t := range g.tags {
		var (
			tagged bool
			item   *gtk.CheckMenuItem
		)

		_, tagged = tags[t.ID]

		if item, err = gtk.CheckMenuItemNewWithLabel(t.Name); err != nil {
			msg = fmt.Sprintf("Cannot create gtk.CheckMenuItem for Tag %s: %s",
				t.Name,
				err.Error())
			goto ERROR
		}

		item.SetActive(tagged)
		menu.Append(item)
	}

	return menu, nil

ERROR:
	g.log.Printf("[ERROR] %s\n", msg)
	g.displayMsg(msg)
	return nil, err
} // func (g *GUI) mkFileTagMenu(f *objects.File) (*gtk.Menu, error)
