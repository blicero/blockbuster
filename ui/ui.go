// /home/krylon/go/src/github.com/blicero/blockbuster/ui/ui.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-06 21:33:38 krylon>

// Package ui provides the user interface for the video library.
package ui

import (
	"log"
	"sync"
	"time"

	"github.com/blicero/blockbuster/common"
	"github.com/blicero/blockbuster/database"
	"github.com/blicero/blockbuster/logdomain"
	"github.com/blicero/blockbuster/tree"
	"github.com/gotk3/gotk3/gtk"
)

const refInterval = time.Second // nolint: deadcode

type tabContent struct {
	store gtk.ITreeModel
	view  *gtk.TreeView
	scr   *gtk.ScrolledWindow
}

// GUI is the ... well, GUI of the application.
// Built using Gtk3, which I am still in the process of learning,
// with the added challenge of using a C library from Go, with the
// added challenge that I basically suck at UI design.
//
// So, you've been warned, if you stick around, some interesting times
// lie ahead!
type GUI struct {
	db       *database.Database
	scan     *tree.Scanner
	log      *log.Logger
	lock     sync.RWMutex // nolint: structcheck,unused
	win      *gtk.Window
	mainBox  *gtk.Box
	menubar  *gtk.MenuBar
	notebook *gtk.Notebook
	tabs     []tabContent
}

// Create creates a new GUI. You didn't see *that* coming, now, did you?
func Create() (*GUI, error) {
	var (
		err error
		g   = new(GUI)
	)

	if g.log, err = common.GetLogger(logdomain.GUI); err != nil {
		return nil, err
	} else if g.db, err = database.Open(common.DbPath); err != nil {
		g.log.Printf("[ERROR] Cannot open Database at %s: %s\n",
			common.DbPath,
			err.Error())
		return nil, err
	} else if g.scan, err = tree.NewScanner(4); err != nil {
		g.log.Printf("[ERROR] Cannot create Scanner: %s\n",
			err.Error())
		return nil, err
	}

	gtk.Init(nil)

	if g.win, err = gtk.WindowNew(gtk.WINDOW_TOPLEVEL); err != nil {
		g.log.Printf("[ERROR] Cannot create Toplevel Window: %s\n",
			err.Error())
		return nil, err
	} else if g.mainBox, err = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 1); err != nil {
		g.log.Printf("[ERROR] Cannot create Box: %s\n",
			err.Error())
		return nil, err
	} else if g.menubar, err = gtk.MenuBarNew(); err != nil {
		g.log.Printf("[ERROR] Cannot create MenuBar: %s\n",
			err.Error())
		return nil, err
	} else if g.notebook, err = gtk.NotebookNew(); err != nil {
		g.log.Printf("[ERROR] Cannot create Notebook: %s\n",
			err.Error())
		return nil, err
	}

	var (
		fileMenu                   *gtk.Menu
		scanItem, quitItem, fmItem *gtk.MenuItem
	)

	if fileMenu, err = gtk.MenuNew(); err != nil {
		g.log.Printf("[ERROR] Cannot create File menu: %s\n",
			err.Error())
		return nil, err
	} else if scanItem, err = gtk.MenuItemNewWithMnemonic("_Scan"); err != nil {
		g.log.Printf("[ERROR] Cannot create menu item File/Scan: %s\n",
			err.Error())
		return nil, err
	} else if quitItem, err = gtk.MenuItemNewWithMnemonic("_Quit"); err != nil {
		g.log.Printf("[ERROR] Cannot create menu item File/Quit: %s\n",
			err.Error())
		return nil, err
	} else if fmItem, err = gtk.MenuItemNewWithMnemonic("_File"); err != nil {
		g.log.Printf("[ERROR] Cannot create menu item File/: %s\n",
			err.Error())
		return nil, err
	}

	scanItem.Connect("activate", g.promptScanFolder)
	quitItem.Connect("activate", gtk.MainQuit)

	fmItem.SetSubmenu(fileMenu)

	fileMenu.Append(scanItem)
	fileMenu.Append(quitItem)

	g.menubar.Append(fmItem)

	g.tabs = make([]tabContent, len(viewList))

	for tabIdx, v := range viewList {
		var (
			tab tabContent
			lbl *gtk.Label
		)

		if tab.store, tab.view, err = v.create(); err != nil {
			g.log.Printf("[ERROR] Cannot create TreeView %q: %s\n",
				v.title,
				err.Error())
			return nil, err
		} else if tab.scr, err = gtk.ScrolledWindowNew(nil, nil); err != nil {
			g.log.Printf("[ERROR] Cannot create ScrolledWindow for %q: %s\n",
				v.title,
				err.Error())
			return nil, err
		} else if lbl, err = gtk.LabelNew(v.title); err != nil {
			g.log.Printf("[ERROR] Cannot create title Label for %q: %s\n",
				v.title,
				err.Error())
			return nil, err
		}

		g.tabs[tabIdx] = tab
		tab.scr.Add(tab.view)
		g.notebook.AppendPage(tab.scr, lbl)

	}

	g.win.Connect("destroy", gtk.MainQuit)

	g.mainBox.PackStart(g.menubar, false, false, 0)
	g.mainBox.PackStart(g.notebook, false, false, 0)
	g.win.Add(g.mainBox)
	g.win.SetSizeRequest(960, 540)

	return g, nil
} // func Create() (*GUI, error)

// ShowAndRun displays the GUI and runs the Gtk event loop.
func (g *GUI) ShowAndRun() {
	g.win.ShowAll()
	// glib.TimeoutAdd(uint(refInterval.Milliseconds()), g.renderModelHandler)
	// glib.TimeoutAdd(uint(refInterval.Milliseconds()), g.clearWarningsHandler)
	// go g.scanLoop()
	gtk.Main()
} // func (g *GUI) ShowAndRun()

// nolint: gosimple,unused
func (g *GUI) scanLoop() {
	var ticker = time.NewTicker(refInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			g.log.Println("[TRACE] scanLoop says Hello.")
		}
	}
} // func (g *GUI) scanLoop()

func (g *GUI) promptScanFolder() {
	g.log.Printf("[DEBUG] Suck it!\n")
} // func (g *GUI) promptScanFolder()
