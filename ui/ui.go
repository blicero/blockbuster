// /home/krylon/go/src/github.com/blicero/blockbuster/ui/ui.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-06 15:06:55 krylon>

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
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

const refInterval = time.Second // nolint: deadcode

// nolint: deadcode
const (
	cfTitle = iota
	cfSize
	cfYear
	cfDirector
	cfActors
	cfTags
)

// nolint: deadcode
const (
	cdDirector = iota
	cdTitle
	cdYear
	cdActors
	cdTags
)

// nolint: deadcode
const (
	ctName = iota
	ctTitle
	ctYear
)

// GUI is the ... well, GUI of the application.
// Built using Gtk3, which I am still in the process of learning,
// with the added challenge of using a C library from Go, with the
// added challenge that I basically suck at UI design.
//
// So, you've been warned, if you stick around, some interesting times
// lie ahead!
type GUI struct {
	db        *database.Database
	scan      *tree.Scanner
	log       *log.Logger
	lock      sync.RWMutex // nolint: structcheck,unused
	win       *gtk.Window
	mainBox   *gtk.Box
	menubar   *gtk.MenuBar
	fileScr   *gtk.ScrolledWindow
	dirScr    *gtk.ScrolledWindow
	tagScr    *gtk.ScrolledWindow
	notebook  *gtk.Notebook
	fileView  *gtk.TreeView
	dirView   *gtk.TreeView
	tagView   *gtk.TreeView
	fileStore *gtk.ListStore
	dirStore  *gtk.TreeStore
	tagStore  *gtk.TreeStore
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
	} else if g.fileStore, err = gtk.ListStoreNew(
		glib.TYPE_STRING, // Title
		glib.TYPE_STRING, // Size
		glib.TYPE_INT,    // Year
		glib.TYPE_STRING, // Director
		glib.TYPE_STRING, // Actor(s)
		glib.TYPE_STRING, // Tags
	); err != nil {
		g.log.Printf("[ERROR] Cannot create fileStore: %s\n",
			err.Error())
		return nil, err
	} else if g.fileView, err = gtk.TreeViewNewWithModel(g.fileStore); err != nil {
		g.log.Printf("[ERROR] Cannot create fileView: %s\n",
			err.Error())
		return nil, err
	} else if g.fileScr, err = gtk.ScrolledWindowNew(nil, nil); err != nil {
		g.log.Printf("[ERROR] Cannot create ScrolledWindow for fileView: %s\n",
			err.Error())
		return nil, err
	} else if g.dirStore, err = gtk.TreeStoreNew(
		glib.TYPE_STRING, // Director
		glib.TYPE_STRING, // Title
		glib.TYPE_INT,    // Year
		glib.TYPE_STRING, // Actors
		glib.TYPE_STRING, // Tags
	); err != nil {
		g.log.Printf("[ERROR] Cannot create dirStore: %s\n",
			err.Error())
		return nil, err
	} else if g.dirView, err = gtk.TreeViewNewWithModel(g.dirStore); err != nil {
		g.log.Printf("[ERROR] Cannot create dirView: %s\n",
			err.Error())
		return nil, err
	} else if g.dirScr, err = gtk.ScrolledWindowNew(nil, nil); err != nil {
		g.log.Printf("[ERROR] Cannot create ScrolledWindow for dirView: %s\n",
			err.Error())
		return nil, err
	} else if g.tagStore, err = gtk.TreeStoreNew(
		glib.TYPE_STRING, // Name
		glib.TYPE_STRING, // Title
		glib.TYPE_INT,    // Year
	); err != nil {
		g.log.Printf("[ERROR] Cannot create tagStore: %s\n",
			err.Error())
		return nil, err
	} else if g.tagView, err = gtk.TreeViewNewWithModel(g.tagStore); err != nil {
		g.log.Printf("[ERROR] Cannot create tagView: %s\n",
			err.Error())
		return nil, err
	} else if g.tagScr, err = gtk.ScrolledWindowNew(nil, nil); err != nil {
		g.log.Printf("[ERROR] Cannot create ScrolledWindow for tagView: %s\n",
			err.Error())
		return nil, err
	}

	var cTitle, cSize, cYear, cDir, cAct, cTag *gtk.TreeViewColumn

	if cTitle, err = createCol("Title", cfTitle); err != nil {
		g.log.Printf("[ERROR] Cannot create column cfTitle: %s\n",
			err.Error())
		return nil, err
	} else if cSize, err = createCol("Size", cfSize); err != nil {
		g.log.Printf("[ERROR] Cannot create column cfSize: %s\n",
			err.Error())
		return nil, err
	} else if cYear, err = createCol("Year", cfYear); err != nil {
		g.log.Printf("[ERROR] Cannot create column cfYear: %s\n",
			err.Error())
		return nil, err
	} else if cDir, err = createCol("Director", cfDirector); err != nil {
		g.log.Printf("[ERROR] Cannot create column cfDirector: %s\n",
			err.Error())
		return nil, err
	} else if cAct, err = createCol("Actor(s)", cfActors); err != nil {
		g.log.Printf("[ERROR] Cannot create column cfActors: %s\n",
			err.Error())
		return nil, err
	} else if cTag, err = createCol("Tags", cfTags); err != nil {
		g.log.Printf("[ERROR] Cannot create column cfTags: %s\n",
			err.Error())
		return nil, err
	}

	g.fileView.AppendColumn(cTitle)
	g.fileView.AppendColumn(cSize)
	g.fileView.AppendColumn(cYear)
	g.fileView.AppendColumn(cDir)
	g.fileView.AppendColumn(cAct)
	g.fileView.AppendColumn(cTag)

	var colDDir, colDTitle, colDYear, colDActors, colDTags *gtk.TreeViewColumn

	if colDDir, err = createCol("Director", cdDirector); err != nil {
		g.log.Printf("[ERROR] Cannot create column cdDirector: %s\n",
			err.Error())
		return nil, err
	} else if colDTitle, err = createCol("Title", cdTitle); err != nil {
		g.log.Printf("[ERROR] Cannot create column cdTitle: %s\n",
			err.Error())
		return nil, err
	} else if colDYear, err = createCol("Year", cdYear); err != nil {
		g.log.Printf("[ERROR] Cannot create column cdYear: %s\n",
			err.Error())
		return nil, err
	} else if colDActors, err = createCol("Actors", cdActors); err != nil {
		g.log.Printf("[ERROR] Cannot create column cfYear: %s\n",
			err.Error())
		return nil, err
	} else if colDTags, err = createCol("Tags", cdTags); err != nil {
		g.log.Printf("[ERROR] Cannot create column cdTags: %s\n",
			err.Error())
		return nil, err
	}

	g.dirView.AppendColumn(colDDir)
	g.dirView.AppendColumn(colDTitle)
	g.dirView.AppendColumn(colDYear)
	g.dirView.AppendColumn(colDActors)
	g.dirView.AppendColumn(colDTags)

	var colTName, colTTitle, colTYear *gtk.TreeViewColumn

	if colTName, err = createCol("Tag", ctName); err != nil {
		g.log.Printf("[ERROR] Cannot create column ctName: %s\n",
			err.Error())
		return nil, err
	} else if colTTitle, err = createCol("Title", ctTitle); err != nil {
		g.log.Printf("[ERROR] Cannot create column ctTitle: %s\n",
			err.Error())
		return nil, err
	} else if colTYear, err = createCol("Year", ctYear); err != nil {
		g.log.Printf("[ERROR] Cannot create column ctYear: %s\n",
			err.Error())
		return nil, err
	}

	g.tagView.AppendColumn(colTName)
	g.tagView.AppendColumn(colTTitle)
	g.tagView.AppendColumn(colTYear)

	var fLbl, dLbl, tLbl *gtk.Label

	if fLbl, err = gtk.LabelNew("File"); err != nil {
		g.log.Printf("[ERROR] Cannot create Label 'Title': %s\n",
			err.Error())
		return nil, err
	} else if dLbl, err = gtk.LabelNew("Director"); err != nil {
		g.log.Printf("[ERROR] Cannot create Label 'Director': %s\n",
			err.Error())
		return nil, err
	} else if tLbl, err = gtk.LabelNew("Tags"); err != nil {
		g.log.Printf("[ERROR] Cannot create Label 'Tags': %s\n",
			err.Error())
		return nil, err
	}

	g.notebook.AppendPage(g.fileScr, fLbl)
	g.notebook.AppendPage(g.dirScr, dLbl)
	g.notebook.AppendPage(g.tagScr, tLbl)

	g.win.Connect("destroy", gtk.MainQuit)

	g.fileScr.Add(g.fileView)
	g.dirScr.Add(g.dirView)
	g.tagScr.Add(g.tagView)
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
