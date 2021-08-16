// /home/krylon/go/src/github.com/blicero/blockbuster/ui/person.go
// -*- mode: go; coding: utf-8; -*-
// Created on 14. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-16 19:07:57 krylon>

package ui

import (
	"fmt"
	"net/url"
	"os/exec"

	"github.com/blicero/blockbuster/objects"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func (g *GUI) loadPeople() bool {
	g.clearData(tiPerson)

	var (
		err    error
		msg    string
		piter  *gtk.TreeIter
		people []objects.Person
		store  *gtk.TreeStore
	)

	store = g.tabs[tiPerson].store.(*gtk.TreeStore)

	store.Clear()

	if people, err = g.db.PersonGetAll(); err != nil {
		msg = fmt.Sprintf("Database.PersonGetAll failed: %s",
			err.Error())
		goto ERROR
	}

	for pidx := range people {
		var (
			files []objects.File
			p     = &people[pidx]
		)

		// First, we add the Person to the TreeModel.
		piter = store.Append(nil)

		store.SetValue(piter, 0, p.ID)              // nolint: errcheck
		store.SetValue(piter, 1, p.Name)            // nolint: errcheck
		store.SetValue(piter, 2, p.Birthday.Year()) // nolint: errcheck

		if files, err = g.db.ActorGetByPerson(p); err != nil {
			msg = fmt.Sprintf("Cannot load Files with acting credits by %s (%d): %s",
				p.Name,
				p.ID,
				err.Error())
			goto ERROR
		}

		for _, f := range files {
			var fiter = store.Append(piter)

			// Is this a good idea?
			store.SetValue(fiter, 0, f.ID)             // nolint: errcheck
			store.SetValue(fiter, 1, "")               // nolint: errcheck
			store.SetValue(fiter, 3, f.DisplayTitle()) // nolint: errcheck
		}
	}

	return false

ERROR:
	g.log.Printf("[ERROR] %s\n", msg)
	g.displayMsg(msg)
	return false
} // func (g *GUI) loadPeople() bool

func (g *GUI) handlePersonListClick(view *gtk.TreeView, evt *gdk.Event) {
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
		menu   *gtk.Menu
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
		val  *glib.Value
		gval interface{}
		name string
		id   int64
	)

	if val, err = model.GetValue(iter, 0); err != nil {
		msg = fmt.Sprintf("Cannot get ID from column 0: %s",
			err.Error())
		goto ERROR
	} else if gval, err = val.GoValue(); err != nil {
		msg = fmt.Sprintf("Cannot get go value for ID: %s",
			err.Error())
		goto ERROR
	}

	id = int64(gval.(int))

	// First, we need to figure out if the user clicked on a Person or a File.
	if val, err = model.GetValue(iter, 1); err != nil {
		msg = fmt.Sprintf("Cannot get value for column 1: %s",
			err.Error())
		goto ERROR
	} else if name, err = val.GetString(); err != nil {
		msg = fmt.Sprintf("Cannot get string value for column 1: %s",
			err.Error())
		goto ERROR
	}

	if name == "" {
		// File
		g.log.Printf("[DEBUG] Looking at File #%d\n", id)
		var f *objects.File

		if f, err = g.db.FileGetByID(id); err != nil {
			msg = fmt.Sprintf("Cannot lookup File #%d: %s",
				id,
				err.Error())
			goto ERROR
		} else if menu, err = g.mkPersonFileContextMenu(path, f); err != nil {
			msg = fmt.Sprintf("Cannot create context menu for %s: %s",
				f.DisplayTitle(),
				err.Error())
			goto ERROR
		}
	} else {
		// Person
		g.log.Printf("[DEBUG] Looking at Person #%d\n", id)
		var p *objects.Person

		if p, err = g.db.PersonGetByID(id); err != nil {
			msg = fmt.Sprintf("Cannot lookup Person #%d: %s",
				id,
				err.Error())
			goto ERROR
		} else if menu, err = g.mkPersonContextMenu(path, p); err != nil {
			msg = fmt.Sprintf("Cannot create context Menu for %s: %s",
				p.Name,
				err.Error())
			goto ERROR
		}
	}

	menu.ShowAll()
	menu.PopupAtPointer(evt)

	return

ERROR:
	g.log.Printf("[ERROR] %s\n", msg)
	g.displayMsg(msg)
} // func (g *GUI) handlePersonListClick(view *gtk.TreeView, evt *gdk.Event)

func (g *GUI) mkPersonContextMenu(path *gtk.TreePath, p *objects.Person) (*gtk.Menu, error) {
	var (
		err                              error
		menu, urlMenu                    *gtk.Menu
		itemDel, itemURLAdd, itemURLList *gtk.MenuItem
	)

	// It would be nice if I could skip displaying the URL list submenu
	// if there are not URLs linked to the Person.

	if menu, err = gtk.MenuNew(); err != nil {
		return nil, err
	} else if itemDel, err = gtk.MenuItemNewWithMnemonic("_Delete"); err != nil {
		return nil, err
	} else if itemURLAdd, err = gtk.MenuItemNewWithMnemonic("_Add URL"); err != nil {
		return nil, err
	} else if itemURLList, err = gtk.MenuItemNewWithMnemonic("_URLs"); err != nil {
		return nil, err
	} else if urlMenu, err = g.getPersonLinks(p); err != nil {
		return nil, err
	}

	itemURLList.SetSubmenu(urlMenu)
	itemURLAdd.Connect("activate", g.mkPersonAddURLHandler(p))

	menu.Append(itemURLAdd)
	menu.Append(itemURLList)
	menu.Append(itemDel)

	itemURLList.SetSubmenu(urlMenu)

	return menu, nil
} // func (g *GUI) mkPersonContextMenu(path *gtk.TreePath, p *objects.Person) (*gtk.Menu, error)

func (g *GUI) getPersonLinks(p *objects.Person) (*gtk.Menu, error) {
	var (
		err   error
		msg   string
		menu  *gtk.Menu
		links []objects.Link
	)

	if links, err = g.db.PersonURLGetByPerson(p); err != nil {
		msg = fmt.Sprintf("Cannot get Links for %s: %s",
			p.Name,
			err.Error())
		goto ERROR
	} else if menu, err = gtk.MenuNew(); err != nil {
		msg = fmt.Sprintf("Cannot create URL menu for %s: %s",
			p.Name,
			err.Error())
		goto ERROR
	}

	for lidx := range links {
		var (
			item *gtk.MenuItem
			l    = &links[lidx]
		)

		if item, err = gtk.MenuItemNewWithLabel(l.DisplayTitle()); err != nil {
			msg = fmt.Sprintf("Cannot create menu handler for URL %q: %s",
				l.DisplayTitle(),
				err.Error())
			goto ERROR
		}

		item.Connect("activate", g.mkURLHandler(l))
		menu.Append(item)
	}

	return menu, nil

ERROR:
	g.log.Printf("[ERROR] %s\n", msg)
	g.displayMsg(msg)
	return nil, err
} // func (g *GUI) getPersonLinks(p *objects.Person) ([]*gtk.MenuItem, error)

func (g *GUI) mkURLHandler(l *objects.Link) func() {
	const urlOpenCmd = "xdg-open"
	return func() {
		var (
			err error
			cmd *exec.Cmd
		)

		cmd = exec.Command(urlOpenCmd, l.URL.String())
		if err = cmd.Run(); err != nil {
			var msg = fmt.Sprintf("Cannot open URL %s (%s): %s",
				l.DisplayTitle(),
				l.URL.String(),
				err.Error())
			g.log.Printf("[ERROR] %s\n", msg)
			g.displayMsg(msg)
		} else {
			g.log.Printf("[TRACE] Executed %s %s without error, so if it didn't open, it's not my fault.\n",
				urlOpenCmd,
				l.URL.String())
		}
	}
}

func (g *GUI) mkPersonFileContextMenu(path *gtk.TreePath, f *objects.File) (*gtk.Menu, error) {
	var (
		err      error
		msg      string
		menu     *gtk.Menu
		playItem *gtk.MenuItem
	)

	if menu, err = gtk.MenuNew(); err != nil {
		msg = fmt.Sprintf("Cannot create context Menu for File %s: %s",
			f.DisplayTitle(),
			err.Error())
		goto ERROR
	} else if playItem, err = gtk.MenuItemNewWithMnemonic("_Play"); err != nil {
		msg = fmt.Sprintf("Cannot create context menu item to play %s: %s",
			f.DisplayTitle(),
			err.Error())
		goto ERROR
	}

	playItem.Connect("activate", func() { g.playFile(f) })
	menu.Append(playItem)

	return menu, nil

ERROR:
	g.log.Printf("[ERROR] %s\n", msg)
	// g.displayMsg(msg)
	return nil, err
} // func (g *GUI) mkPersonFileContextMenu(path *gtk.TreePath, f *objects.File) (*gtk.Menu, error)

func (g *GUI) mkPersonAddURLHandler(p *objects.Person) func() {
	return func() {
		var (
			err                    error
			s                      string
			l                      objects.Link
			dlg                    *gtk.Dialog
			dbox                   *gtk.Box
			grid                   *gtk.Grid
			uLbl, tLbl, dLbl       *gtk.Label
			uEntry, tEntry, dEntry *gtk.Entry
		)

		if dlg, err = gtk.DialogNewWithButtons(
			"Add URL",
			g.win,
			gtk.DIALOG_MODAL,
			[]interface{}{
				"_Cancel",
				gtk.RESPONSE_CANCEL,
				"_OK",
				gtk.RESPONSE_OK,
			},
		); err != nil {
			g.log.Printf("[ERROR] Cannot create Dialog for adding URL: %s\n",
				err.Error())
			return
		}

		defer dlg.Close()

		if _, err = dlg.AddButton("OK", gtk.RESPONSE_OK); err != nil {
			g.log.Printf("[ERROR] Cannot add cancel button to AddPerson Dialog: %s\n",
				err.Error())
			return
		} else if grid, err = gtk.GridNew(); err != nil {
			g.log.Printf("[ERROR] Cannot create gtk.Grid for AddPerson Dialog: %s\n",
				err.Error())
			return
		} else if uLbl, err = gtk.LabelNew("URL:"); err != nil {
			g.log.Printf("[ERROR] Cannot create URL Label: %s\n",
				err.Error())
			return
		} else if tLbl, err = gtk.LabelNew("Title:"); err != nil {
			g.log.Printf("[ERROR] Cannot create Title Label: %s\n",
				err.Error())
			return
		} else if dLbl, err = gtk.LabelNew("Description:"); err != nil {
			g.log.Printf("[ERROR] Cannot create Description Label: %s\n",
				err.Error())
			return
		} else if uEntry, err = gtk.EntryNew(); err != nil {
			g.log.Printf("[ERROR] Cannot create Entry for URL: %s\n",
				err.Error())
			return
		} else if tEntry, err = gtk.EntryNew(); err != nil {
			g.log.Printf("[ERROR] Cannot create Entry for Title: %s\n",
				err.Error())
			return
		} else if dEntry, err = gtk.EntryNew(); err != nil {
			g.log.Printf("[ERROR] Cannot create Entry for URL Description: %s\n",
				err.Error())
			return
		} else if dbox, err = dlg.GetContentArea(); err != nil {
			g.log.Printf("[ERROR] Cannot get ContentArea of AddPerson Dialog: %s\n",
				err.Error())
			return
		}

		grid.InsertColumn(0)
		grid.InsertColumn(1)
		grid.InsertRow(0)
		grid.InsertRow(1)
		grid.InsertRow(2)

		grid.Attach(uLbl, 0, 0, 1, 1)
		grid.Attach(tLbl, 0, 1, 1, 1)
		grid.Attach(dLbl, 0, 2, 1, 1)
		grid.Attach(uEntry, 1, 0, 1, 1)
		grid.Attach(tEntry, 1, 1, 1, 1)
		grid.Attach(dEntry, 1, 2, 1, 1)

		dbox.PackStart(grid, true, true, 0)
		dlg.ShowAll()

		var res = dlg.Run()

		switch res {
		case gtk.RESPONSE_NONE:
			fallthrough
		case gtk.RESPONSE_DELETE_EVENT:
			fallthrough
		case gtk.RESPONSE_CLOSE:
			fallthrough
		case gtk.RESPONSE_CANCEL:
			g.log.Println("[DEBUG] User changed their mind about adding a Link. Fine with me.")
			return
		case gtk.RESPONSE_OK:
			// 's ist los, Hund?
		default:
			g.log.Printf("[CANTHAPPEN] Well, I did NOT see this coming: %d\n",
				res)
			return
		}

		if s, err = uEntry.GetText(); err != nil {
			g.log.Printf("[ERROR] Cannot get input from URL field: %s\n",
				err.Error())
			return
		} else if l.Title, err = tEntry.GetText(); err != nil {
			g.log.Printf("[ERROR] Cannot get input from Title field: %s\n",
				err.Error())
			return
		} else if l.Description, err = dEntry.GetText(); err != nil {
			g.log.Printf("[ERROR] Cannot get input from Description field: %s\n",
				err.Error())
			return
		} else if l.URL, err = url.Parse(s); err != nil {
			var msg = fmt.Sprintf("This is not a valid URL: %q",
				s)
			g.log.Printf("[ERROR] %s\n", msg)
			g.displayMsg(msg)
			return
		} else if err = g.db.PersonURLAdd(p, &l); err != nil {
			var msg = fmt.Sprintf("Cannot attach Link %q to %s: %s",
				s,
				p.Name,
				err.Error())
			g.log.Printf("[ERROR] %s\n", msg)
			g.displayMsg(msg)
			return
		}

		g.log.Printf("[DEBUG] Guess what? We *successfully* added the URL %q (%s) to %s\n",
			s,
			l.Title,
			p.Name)
	}
} // func (g *GUI) mkPersonAddURLHandler(p *objects.Person) func()
