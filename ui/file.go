// /home/krylon/go/src/github.com/blicero/blockbuster/ui/file.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-12 00:58:26 krylon>

package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/blicero/blockbuster/objects"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func (g *GUI) mkFileTagMenu(path *gtk.TreePath, f *objects.File) (*gtk.Menu, error) {
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

	for idx, t := range g.tags {
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

		if !tagged {
			item.Connect("activate", g.mkFileTagAddHandler(path, f, &g.tags[idx]))
		} else {
			item.Connect("activate", g.mkFileTagDelHandler(path, f, &g.tags[idx]))
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

func (g *GUI) mkFileTagAddHandler(path *gtk.TreePath, f *objects.File, t *objects.Tag) func() {
	return func() {
		var (
			err error
			msg string
		)

		if err = g.db.TagLinkAdd(f, t); err != nil {
			msg = fmt.Sprintf("Cannot link Tag %s to File %s: %s",
				t.Name,
				f.DisplayTitle(),
				err.Error())
			goto ERROR
		}

		// TODO Update ListStore!!!
		glib.IdleAdd(g.mkFileTagListUpdater(path, f))

		return

	ERROR:
		g.log.Printf("[ERROR] %s\n", msg)
		g.displayMsg(msg)
	}
} // func (g *GUI) mkFileTagAddHandler(path *gtk.TreePath, f *objects.File, t *objects.Tag) func()

func (g *GUI) mkFileTagDelHandler(path *gtk.TreePath, f *objects.File, t *objects.Tag) func() {
	return func() {
		var (
			err error
			msg string
		)

		if err = g.db.TagLinkDelete(f, t); err != nil {
			msg = fmt.Sprintf("Cannot unlink Tag %s from File %s: %s",
				t.Name,
				f.DisplayTitle(),
				err.Error())
			goto ERROR
		}

		// TODO Update ListStore!!!
		glib.IdleAdd(g.mkFileTagListUpdater(path, f))

		return

	ERROR:
		g.log.Printf("[ERROR] %s\n", msg)
		g.displayMsg(msg)
	}
} // func (g *GUI) mkFileTagDelHandler(path *gtk.TreePath, f *objects.File, t *objects.Tag) func()

func (g *GUI) mkFileTagListUpdater(path *gtk.TreePath, f *objects.File) func() bool {
	return func() bool {
		var (
			err       error
			msg, tstr string
			iter      *gtk.TreeIter
			store     *gtk.ListStore
			tags      map[int64]objects.Tag
			tlist     []string
		)

		if tags, err = g.db.TagLinkGetByFile(f); err != nil {
			msg = fmt.Sprintf("Cannot get list of Tags for File %s: %s",
				f.DisplayTitle(),
				err.Error())
			goto ERROR
		}

		tlist = make([]string, 0, len(tags))

		for _, t := range tags {
			tlist = append(tlist, t.Name)
		}

		sort.Strings(tlist)
		tstr = strings.Join(tlist, ", ")

		store = g.tabs[tiFile].store.(*gtk.ListStore)

		if iter, err = store.GetIter(path); err != nil {
			msg = fmt.Sprintf("Cannot get TreeIter for File %s: %s",
				f.DisplayTitle(),
				err.Error())
			goto ERROR
		} else if err = store.Set(iter, []int{6}, []interface{}{tstr}); err != nil {
			msg = fmt.Sprintf("Cannot set Tag list for File %s: %s",
				f.DisplayTitle(),
				err.Error())
			goto ERROR
		}

		return false

	ERROR:
		g.log.Printf("[ERROR] %s\n", msg)
		g.displayMsg(msg)
		return false
	}
} // func mkFileTagListUpdater(path *gtk.TreePath, f *objects.File) func () bool
