// /home/krylon/go/src/github.com/blicero/blockbuster/ui/file.go
// -*- mode: go; coding: utf-8; -*-
// Created on 11. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-18 19:56:14 krylon>

package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/blicero/blockbuster/common"
	"github.com/blicero/blockbuster/objects"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func (g *GUI) handleFileListClick(view *gtk.TreeView, evt *gdk.Event) {
	var be = gdk.EventButtonNewFromEvent(evt)

	if be.Button() != gdk.BUTTON_SECONDARY {
		return
	}

	var (
		err    error
		msg    string
		exists bool
		x, y   float64
		path   *gtk.TreePath
		col    *gtk.TreeViewColumn
		model  *gtk.TreeModel
		imodel gtk.ITreeModel
		iter   *gtk.TreeIter
	)

	x = be.X()
	y = be.Y()

	path, col, _, _, exists = view.GetPathAtPos(int(x), int(y))

	if !exists {
		g.log.Printf("[DEBUG] There is no item at %f/%f\n",
			x,
			y)
		return
	}

	g.log.Printf("[DEBUG] Handle Click at %f/%f -> Path %s\n",
		x,
		y,
		path)

	if imodel, err = view.GetModel(); err != nil {
		g.log.Printf("[ERROR] Cannot get Model from View: %s\n",
			err.Error())
		return
	}

	model = imodel.ToTreeModel()

	if iter, err = model.GetIter(path); err != nil {
		g.log.Printf("[ERROR] Cannot get Iter from TreePath %s: %s\n",
			path,
			err.Error())
		return
	}

	var title string = col.GetTitle()
	g.log.Printf("[DEBUG] Column %s was clicked\n",
		title)

	var (
		val *glib.Value
		gv  interface{}
		id  int64
	)

	if val, err = model.GetValue(iter, 0); err != nil {
		g.log.Printf("[ERROR] Cannot get value for column 0: %s\n",
			err.Error())
		return
	} else if gv, err = val.GoValue(); err != nil {
		g.log.Printf("[ERROR] Cannot get Go value from GLib value: %s\n",
			err.Error())
	}

	switch v := gv.(type) {
	case int:
		id = int64(v)
	case int64:
		id = v
	default:
		g.log.Printf("[ERROR] Unexpected type for ID column: %T\n",
			v)
	}

	g.log.Printf("[DEBUG] ID of clicked-on row is %d\n",
		id)

	var (
		f           *objects.File
		contextMenu *gtk.Menu
	)

	if f, err = g.db.FileGetByID(id); err != nil {
		msg = fmt.Sprintf("Cannot look up File #%d: %s",
			id,
			err.Error())
		goto ERROR
	} else if contextMenu, err = g.mkFileContextMenu(path, f); err != nil {
		msg = fmt.Sprintf("Cannot create File context menu: %s",
			err.Error())
		goto ERROR
	}

	contextMenu.ShowAll()
	contextMenu.PopupAtPointer(evt)
	return

ERROR:
	g.log.Printf("[ERROR] %s\n", msg)
	g.displayMsg(msg)
} // func (g *GUI) handleFileListClick(view *gtk.TreeView, evt *gdk.Event)

func (g *GUI) mkFileContextMenu(path *gtk.TreePath, f *objects.File) (*gtk.Menu, error) {
	var (
		err                           error
		msg                           string
		actItem, tagItem, playItem    *gtk.MenuItem
		hideItem                      *gtk.CheckMenuItem
		contextMenu, tagMenu, actMenu *gtk.Menu
	)

	if contextMenu, err = gtk.MenuNew(); err != nil {
		msg = fmt.Sprintf("Cannot create context menu: %s",
			err.Error())
		goto ERROR
	} else if tagMenu, err = g.mkFileTagMenu(path, f); err != nil {
		msg = fmt.Sprintf("Cannot create submenu Tag: %s",
			err.Error())
		goto ERROR
	} else if actMenu, err = g.mkFileActorMenu(path, f); err != nil {
		msg = fmt.Sprintf("Cannot create submenu Actor: %s",
			err.Error())
		goto ERROR
	} else if actItem, err = gtk.MenuItemNewWithMnemonic("_Actors"); err != nil {
		msg = fmt.Sprintf("Cannot create context menu item Actors: %s",
			err.Error())
		goto ERROR
	} else if tagItem, err = gtk.MenuItemNewWithMnemonic("_Tag"); err != nil {
		msg = fmt.Sprintf("Cannot create context menu item Tag: %s",
			err.Error())
		goto ERROR
	} else if hideItem, err = gtk.CheckMenuItemNewWithLabel("Hide"); err != nil {
		msg = fmt.Sprintf("Cannot create context menu item Hide: %s",
			err.Error())
		goto ERROR
	} else if playItem, err = gtk.MenuItemNewWithMnemonic("_Play"); err != nil {
		msg = fmt.Sprintf("Cannot create context menu item Play: %s",
			err.Error())
		goto ERROR
	}

	playItem.Connect("activate", func() { g.playFile(f) })

	actItem.SetSubmenu(actMenu)
	tagItem.SetSubmenu(tagMenu)

	contextMenu.Append(actItem)
	contextMenu.Append(tagItem)
	contextMenu.Append(hideItem)
	contextMenu.Append(playItem)

	return contextMenu, nil
ERROR:
	g.log.Printf("[ERROR] %s\n", msg)
	// g.displayMsg(msg)
	return nil, err
} // func (g *GUI) mkFileContextMenu(path *gtk.TreePath, f *objects.File) (*gtk.Menu, error)

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

func (g *GUI) mkFileEditHandler(colIdx int) func(*gtk.CellRendererText, string, string) {
	if common.Debug {
		g.log.Printf("[DEBUG] Create FileView edit handler for column %d\n", colIdx)
	}

	return func(r *gtk.CellRendererText, pStr, text string) {
		var (
			err       error
			msg       string
			iter      *gtk.TreeIter
			path      *gtk.TreePath
			year, fid int64
			fidVal    *glib.Value
			val       interface{}
			f         *objects.File
			store     = g.tabs[tiFile].store.(*gtk.ListStore)
		)

		g.log.Printf("[TRACE] FileView edit handler for column %d: %q\n",
			colIdx,
			text)

		if path, err = gtk.TreePathNewFromString(pStr); err != nil {
			msg = fmt.Sprintf("Cannot convert string %q to TreePath: %s",
				pStr,
				err.Error())
			goto ERROR
		} else if iter, err = store.GetIter(path); err != nil {
			msg = fmt.Sprintf("Cannot get TreeIter from TreePath %s: %s",
				path,
				err.Error())
			goto ERROR
		} else if fidVal, err = store.GetValue(iter, 0); err != nil {
			msg = fmt.Sprintf("Cannot get File ID from ListStore: %s",
				err.Error())
			goto ERROR
		} else if val, err = fidVal.GoValue(); err != nil {
			msg = fmt.Sprintf("Cannot extract Go value from ID column: %s",
				err.Error())
			goto ERROR
		}

		fid = int64(val.(int))

		if f, err = g.db.FileGetByID(fid); err != nil {
			msg = fmt.Sprintf("Cannot get File #%d: %s",
				fid,
				err.Error())
			goto ERROR
		}

		// FIXME - Update database, too!
		switch colIdx {
		case 1: // Title
			g.log.Printf("[DEBUG] Edit Title: %q\n", text)
			if err = g.db.FileUpdateTitle(f, text); err != nil {
				msg = fmt.Sprintf("Cannot update Title of File %s (%d): %s",
					f.DisplayTitle(),
					f.ID,
					err.Error())
				goto ERROR
			}
			val = text
		case 3: // Year
			if year, err = strconv.ParseInt(text, 10, 64); err != nil {
				msg = fmt.Sprintf("Cannot parse year %q: %s",
					text,
					err.Error())
				goto ERROR
			} else if err = g.db.FileUpdateYear(f, year); err != nil {
				msg = fmt.Sprintf("Cannot update Year for File %s (%d) to %d: %s",
					f.DisplayTitle(),
					f.ID,
					year,
					err.Error())
				goto ERROR
			}
			g.log.Printf("[DEBUG] Edit Year: %d\n", year)
			val = year
		default:
			msg = fmt.Sprintf("I do not know how to edit File View column #%d", colIdx)
			goto ERROR
		}

		if err = store.Set(iter, []int{colIdx}, []interface{}{val}); err != nil {
			msg = fmt.Sprintf("Error updating ListStore: %s",
				err.Error())
			goto ERROR
		}

		return
	ERROR:
		g.log.Printf("[ERROR] %s\n", msg)
		g.displayMsg(msg)
	}
} // func (g *GUI) mkFileEditHandler(column int) func (r *gtk.CellRendererText, path, text string)
