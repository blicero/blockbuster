// /home/krylon/go/src/github.com/blicero/blockbuster/ui/file.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-13 21:24:49 krylon>

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

		item.SetActive(tagged)
		item.Connect("activate", g.mkFileTagToggleHandler(path, tagged, f, &g.tags[idx]))
		menu.Append(item)
	}

	return menu, nil

ERROR:
	g.log.Printf("[ERROR] %s\n", msg)
	g.displayMsg(msg)
	return nil, err
} // func (g *GUI) mkFileTagMenu(f *objects.File) (*gtk.Menu, error)

func (g *GUI) mkFileTagToggleHandler(path *gtk.TreePath, tagged bool, f *objects.File, t *objects.Tag) func() {
	return func() {
		var (
			err error
			msg string
		)

		if !tagged {
			if err = g.db.TagLinkAdd(f, t); err != nil {
				msg = fmt.Sprintf("Cannot link Tag %s to File %s: %s",
					t.Name,
					f.DisplayTitle(),
					err.Error())
				goto ERROR
			}
		} else if err = g.db.TagLinkDelete(f, t); err != nil {
			msg = fmt.Sprintf("Cannot unlink Tag %s from File %s: %s",
				t.Name,
				f.DisplayTitle(),
				err.Error())
			goto ERROR
		}

		glib.IdleAdd(g.mkFileTagListUpdater(path, f))

		return

	ERROR:
		g.log.Printf("[ERROR] %s\n", msg)
		g.displayMsg(msg)
	}
} // func (g *GUI) mkFileTagToggleHandler(path *gtk.TreePath, f *objects.File, t *objects.Tag) func()

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

func (g *GUI) mkFileActorMenu(path *gtk.TreePath, f *objects.File) (*gtk.Menu, error) {
	var (
		err           error
		msg           string
		actors        map[int64]objects.Person
		alist, people []objects.Person
		menu          *gtk.Menu
	)

	if people, err = g.db.PersonGetAll(); err != nil {
		msg = fmt.Sprintf("Cannot load all people from Database: %s",
			err.Error())
		goto ERROR
	} else if alist, err = g.db.ActorGetByFile(f); err != nil {
		msg = fmt.Sprintf("Cannot load Actors for %s from Database: %s",
			f.DisplayTitle(),
			err.Error())
		goto ERROR
	} else if menu, err = gtk.MenuNew(); err != nil {
		msg = fmt.Sprintf("Cannot create Menu for Actors for %s: %s",
			f.DisplayTitle(),
			err.Error())
		goto ERROR
	}

	actors = make(map[int64]objects.Person, len(alist))

	for _, p := range alist {
		actors[p.ID] = p
	}

	// ...
	for i, p := range people {
		var (
			linked bool
			item   *gtk.CheckMenuItem
		)

		_, linked = actors[p.ID]

		if item, err = gtk.CheckMenuItemNewWithLabel(p.Name); err != nil {
			msg = fmt.Sprintf("Cannot create gtk.CheckMenuItem for Person %s: %s",
				p.Name,
				err.Error())
			goto ERROR
		}

		item.SetActive(linked)
		item.Connect("activate", g.mkFileActorToggleHandler(path, linked, f, &people[i]))
		menu.Append(item)
	}

	return menu, nil

ERROR:
	g.log.Printf("[ERROR] %s\n", msg)
	g.displayMsg(msg)
	return nil, err
} // func (g *GUI) mkFileActorMenu(path *gtk.TreePath, f *objects.File) (*gtk.Menu, error)

func (g *GUI) mkFileActorToggleHandler(path *gtk.TreePath, linked bool, f *objects.File, p *objects.Person) func() {
	return func() {
		var (
			err error
			msg string
		)

		if !linked {
			err = g.db.ActorAdd(f, p)
		} else {
			err = g.db.ActorDelete(f, p)
		}

		if err != nil {
			msg = fmt.Sprintf("Error toggling Actor %s for %s (%t -> %t): %s",
				p.Name,
				f.DisplayTitle(),
				linked,
				!linked,
				err.Error())
			goto ERROR
		}

		glib.IdleAdd(g.mkFileActorListUpdate(path, f))
		if !linked {
			glib.IdleAdd(g.makeNewActorHandler(p, f))
		} else {
			glib.IdleAdd(func() bool {
				g.removeActor(p, f)
				return false
			})
		}
		return

	ERROR:
		g.log.Printf("[ERROR] %s\n", msg)
		g.displayMsg(msg)
	}
} // func (g *GUI) mkFileActorToggleHandler(path *gtk.TreePath, linked bool, f *objects.File, p *objects.Person) func()

func (g *GUI) mkFileActorListUpdate(path *gtk.TreePath, f *objects.File) func() bool {
	return func() bool {
		var (
			err       error
			msg, astr string
			iter      *gtk.TreeIter
			store     *gtk.ListStore
			actors    []objects.Person
			alist     []string
		)

		if actors, err = g.db.ActorGetByFile(f); err != nil {
			msg = fmt.Sprintf("Cannot get Actors for %s: %s",
				f.DisplayTitle(),
				err.Error())
			goto ERROR
		}

		alist = make([]string, len(actors))

		for i, a := range actors {
			alist[i] = a.Name
		}

		astr = strings.Join(alist, ", ")

		store = g.tabs[tiFile].store.(*gtk.ListStore)

		if iter, err = store.GetIter(path); err != nil {
			msg = fmt.Sprintf("Cannot get TreeIter for Actors of %s (%s): %s",
				f.DisplayTitle(),
				path,
				err.Error())
			goto ERROR
		} else if err = store.Set(iter, []int{5}, []interface{}{astr}); err != nil {
			msg = fmt.Sprintf("Error updating Actor list for %s: %s",
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
} // func (g *GUI) mkFileActorListUpdate(path *gtk.TreePath, p *objects.Person) func () bool
