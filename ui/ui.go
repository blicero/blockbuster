// /home/krylon/go/src/github.com/blicero/blockbuster/ui/ui.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-06 23:43:13 krylon>

// Package ui provides the user interface for the video library.
package ui

import (
	"log"
	"sync"
	"time"

	"github.com/blicero/blockbuster/common"
	"github.com/blicero/blockbuster/database"
	"github.com/blicero/blockbuster/logdomain"
	"github.com/blicero/blockbuster/objects"
	"github.com/blicero/blockbuster/tree"
	"github.com/gotk3/gotk3/gtk"
)

const (
	qDepth      = 128
	refInterval = time.Second // nolint: deadcode
)

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
	scanner  *tree.Scanner
	log      *log.Logger
	fileQ    chan *objects.File
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
	} else if g.scanner, err = tree.NewScanner(4); err != nil {
		g.log.Printf("[ERROR] Cannot create Scanner: %s\n",
			err.Error())
		return nil, err
	}

	g.fileQ = make(chan *objects.File, qDepth)

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
	var err error

	if err = g.scanner.Start(g.fileQ); err != nil {
		g.log.Printf("[ERROR] Cannot start Scanner: %s\n",
			err.Error())
		return
	}

	go g.scanLoop()

	g.win.ShowAll()
	gtk.Main()
} // func (g *GUI) ShowAndRun()

func (g *GUI) scanLoop() {
	var ticker = time.NewTicker(refInterval)
	defer ticker.Stop()

	for g.scanner.Active() {
		select {
		case <-ticker.C:
			//g.log.Println("[TRACE] scanLoop says Hello.")
			continue
		case f := <-g.fileQ:
			g.log.Printf("[DEBUG] Received new File %d: %s\n",
				f.ID,
				f.Path)
		}
	}
} // func (g *GUI) scanLoop()

func (g *GUI) promptScanFolder() {
	g.log.Printf("[DEBUG] You scannin', or what?!\n")
	var (
		err error
		dlg *gtk.FileChooserDialog
		res gtk.ResponseType
	)

	if dlg, err = gtk.FileChooserDialogNewWith2Buttons(
		"Scan Folder",
		g.win,
		gtk.FILE_CHOOSER_ACTION_SELECT_FOLDER,
		"Cancel",
		gtk.RESPONSE_CANCEL,
		"OK",
		gtk.RESPONSE_OK,
	); err != nil {
		g.log.Printf("[ERROR] Cannot create FileChooserDialog: %s\n",
			err.Error())
		return
	}

	defer dlg.Close()

	res = dlg.Run()

	switch res {
	case gtk.RESPONSE_CANCEL:
		g.log.Println("[DEBUG] Ha, you almost got me.")
		return
	case gtk.RESPONSE_OK:
		var path string
		if path, err = dlg.GetCurrentFolder(); err != nil {
			g.log.Printf("[ERROR] Cannot folder selected by user: %s\n",
				err.Error())
			return
		}

		g.log.Printf("[DEBUG] Telling Scanner to visit %s\n",
			path)

		g.scanner.ScanPath(path)
	}

} // func (g *GUI) promptScanFolder()
