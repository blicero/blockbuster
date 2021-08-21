// /home/krylon/go/src/github.com/blicero/blockbuster/ui/tag.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-21 20:55:32 krylon>

package ui

import (
	"fmt"
	"sort"

	"github.com/blicero/blockbuster/common"
	"github.com/blicero/blockbuster/objects"
	"github.com/blicero/krylib"
	"github.com/gotk3/gotk3/gtk"
)

func (g *GUI) tagAdd(t *objects.Tag) error {
	krylib.Trace()
	var (
		err   error
		model *gtk.TreeStore
		iter  *gtk.TreeIter
	)

	model = g.tabs[tiTags].store.(*gtk.TreeStore)

	iter = model.Append(nil)

	if err = model.SetValue(iter, 0, t.ID); err != nil {
		g.log.Printf("[ERROR] Cannot set ID for Tag %s: %s\n",
			t.Name,
			err.Error())
		return err
	} else if err = model.SetValue(iter, 1, t.Name); err != nil {
		g.log.Printf("[ERROR] Cannot set Name for Tag %s: %s\n",
			t.Name,
			err.Error())
		return err
	}

	g.tags = append(g.tags, *t)
	sort.Sort(g.tags)

	return nil
} // func (g *GUI) tagAdd(t *objects.Tag) error

func (g *GUI) loadTagView() bool {
	krylib.Trace()
	var (
		err   error
		msg   string
		tags  []objects.Tag
		store *gtk.TreeStore
	)

	if tags, err = g.db.TagGetAll(); err != nil {
		msg = fmt.Sprintf("Cannot load all Tags from Database: %s",
			err.Error())
		goto ERROR
	}

	store = g.tabs[tiTags].store.(*gtk.TreeStore)
	store.Clear() // ???

	for tidx := range tags {
		var (
			files []objects.File
			titer *gtk.TreeIter
			t     = &tags[tidx]
		)

		if files, err = g.db.TagLinkGetByTag(t); err != nil {
			msg = fmt.Sprintf("Failed to load Files linked to Tag %s: %s",
				t.Name,
				err.Error())
			goto ERROR
		}

		titer = store.Append(nil)
		store.SetValue(titer, 0, t.ID)   // nolint: errcheck
		store.SetValue(titer, 1, t.Name) // nolint: errcheck

		for fidx := range files {
			var (
				f     = &files[fidx]
				fiter = store.Append(titer)
			)

			store.SetValue(fiter, 2, f.DisplayTitle()) // nolint: errcheck
			store.SetValue(fiter, 3, int(f.Year))      // nolint: errcheck
		}
	}

	return false
ERROR:
	if !common.Debug {
		store.Clear()
	}
	g.log.Printf("[ERROR] %s\n", msg)
	g.displayMsg(msg)
	return false
} // func (g *GUI) loadTagView() bool
