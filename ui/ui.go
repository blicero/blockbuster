// /home/krylon/go/src/github.com/blicero/blockbuster/ui/ui.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-11 23:18:21 krylon>

// Package ui provides the user interface for the video library.
package ui

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/blicero/blockbuster/common"
	"github.com/blicero/blockbuster/database"
	"github.com/blicero/blockbuster/logdomain"
	"github.com/blicero/blockbuster/objects"
	"github.com/blicero/blockbuster/tree"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
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
	tags     objects.TagList
}

// Create creates a new GUI. You didn't see *that* coming, now, did you?
func Create() (*GUI, error) {
	var (
		err error
		g   = &GUI{
			fileQ: make(chan *objects.File, qDepth),
		}
	)

	if g.log, err = common.GetLogger(logdomain.GUI); err != nil {
		return nil, err
	} else if g.db, err = database.Open(common.DbPath); err != nil {
		g.log.Printf("[ERROR] Cannot open Database at %s: %s\n",
			common.DbPath,
			err.Error())
		return nil, err
	} else if g.scanner, err = tree.NewScanner(g.fileQ); err != nil {
		g.log.Printf("[ERROR] Cannot create Scanner: %s\n",
			err.Error())
		return nil, err
	} else if g.tags, err = g.db.TagGetAll(); err != nil {
		g.log.Printf("[ERROR] Cannot fetch all Tags from Database: %s\n",
			err.Error())
		return nil, err
	}

	sort.Sort(g.tags)

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

	if err = g.initMenu(); err != nil {
		g.log.Printf("[ERROR] Failed to create menu: %s\n",
			err.Error())
		return nil, err
	}

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

	////////////////////////////////////////////////////////////////////////////////
	///// Context Menus ////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////

	// One thing I really liked about the old Ruby application was that I
	// had sensible context menus, so I could right-click any object in a
	// tree view and get a meaningful list of things to do.

	g.tabs[tiFile].view.Connect("button-press-event", g.handleFileListClick)
	// g.tabs[tiFolder].view.Connect("button-press-event", g.handleFileListClick)

	g.win.Connect("destroy", gtk.MainQuit)

	g.mainBox.PackStart(g.menubar, false, false, 0)
	g.mainBox.PackStart(g.notebook, true, true, 0)
	g.win.Add(g.mainBox)
	g.win.SetSizeRequest(960, 540)

	return g, nil
} // func Create() (*GUI, error)

// ShowAndRun displays the GUI and runs the Gtk event loop.
func (g *GUI) ShowAndRun() {
	var (
		err        error
		fileList   []objects.File
		folderList []objects.Folder
	)

	if fileList, err = g.db.FileGetAll(); err != nil {
		g.log.Printf("[ERROR] Cannot get list of all Files: %s\n",
			err.Error())
		return
	}

	for idx := range fileList {
		var handler = g.makeNewFileHandler(&fileList[idx])
		glib.IdleAdd(handler)
	}

	if folderList, err = g.db.FolderGetAll(); err != nil {
		g.log.Printf("[ERROR] Cannot get list of all Folders: %s\n",
			err.Error())
		return
	}

	for idx := range folderList {
		var handler = g.makeNewFolderHandler(&folderList[idx])
		glib.IdleAdd(handler)
	}

	go g.scanLoop()

	g.win.ShowAll()
	gtk.Main()
} // func (g *GUI) ShowAndRun()

func (g *GUI) scanLoop() {
	var ticker = time.NewTicker(refInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			//g.log.Println("[TRACE] scanLoop says Hello.")
			continue
		case f := <-g.fileQ:
			g.log.Printf("[DEBUG] Received new File %d: %s\n",
				f.ID,
				f.Path)
			glib.IdleAdd(g.makeNewFileHandler(f))
		}
	}
} // func (g *GUI) scanLoop()

func (g *GUI) makeNewFileHandler(f *objects.File) func() bool {
	var store *gtk.ListStore

	switch t := g.tabs[tiFile].store.(type) {
	case *gtk.ListStore:
		store = t
	default:
		g.log.Printf("[CANTHAPPEN] Unexpected type for g.tabs[tiFile].store: %T (expected *gtk.ListStore)\n",
			g.tabs[tiFile].store)
		return func() bool { return false }
	}

	return func() bool {
		var (
			err  error
			iter = store.Append()
		)

		if err = store.Set(
			iter,
			[]int{0, 1},
			[]interface{}{f.ID, f.Path},
		); err != nil {
			g.log.Printf("[ERROR] Cannot add File %d (%s) to Store: %s\n",
				f.ID,
				f.Path,
				err.Error())
		}

		return false
	}
} // func (g *GUI) makeNewFileHandler(f *objects.File) func() bool

func (g *GUI) makeNewFolderHandler(f *objects.Folder) func() bool {
	var store *gtk.ListStore

	switch t := g.tabs[tiFolder].store.(type) {
	case *gtk.ListStore:
		store = t
	default:
		g.log.Printf("[CANTHAPPEN] Unexpected type for g.tabs[tiFolder].store: %T (expected *gtk.ListStore)\n",
			g.tabs[tiFolder].store)
		return func() bool { return false }
	}

	return func() bool {
		var (
			err  error
			iter = store.Append()
		)

		if err = store.Set(
			iter,
			[]int{0, 1},
			[]interface{}{f.ID, f.Path},
		); err != nil {
			g.log.Printf("[ERROR] Cannot add FOlder %d (%s) to Store: %s\n",
				f.ID,
				f.Path,
				err.Error())
		}

		return false
	}
} // func (g *GUI) makeNewFolderHandler(f *objects.Folder) func() bool

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
		f                    *objects.File
		contextMenu, tagMenu *gtk.Menu
		tagItem, playItem    *gtk.MenuItem
		hideItem             *gtk.CheckMenuItem
	)

	if f, err = g.db.FileGetByID(id); err != nil {
		msg = fmt.Sprintf("Cannot look up File #%d: %s",
			id,
			err.Error())
		goto ERROR
	} else if contextMenu, err = gtk.MenuNew(); err != nil {
		msg = fmt.Sprintf("Cannot create context menu: %s",
			err.Error())
		goto ERROR
	} else if tagMenu, err = g.mkFileTagMenu(path, f); err != nil {
		msg = fmt.Sprintf("Cannot create submenu Tag: %s",
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

	tagItem.SetSubmenu(tagMenu)

	contextMenu.Append(tagItem)
	contextMenu.Append(hideItem)
	contextMenu.Append(playItem)

	contextMenu.ShowAll()

	contextMenu.PopupAtPointer(evt)

	return
ERROR:
	g.log.Printf("[ERROR] %s\n", msg)
	g.displayMsg(msg)
} // func (g *GUI) handleFileListClick(view *gtk.TreeView, evt *gdk.Event)

func (g *GUI) handleTagAdd() {
	var (
		err        error
		dlg        *gtk.Dialog
		dbox, hbox *gtk.Box
		lbl        *gtk.Label
		entry      *gtk.Entry
	)

	// XXX I would naively assume that if the function gtk.DialogNewWithButtons
	//     accepts a slice of buttons (actually, pairs of strings and response values),
	//     that it would display them.
	//     But in my tests, I have only ever had one button displayed if they were supplied
	//     via DialogNewWithButtons. No problemo, I thought, I can supply an empty slice and
	//     use Dialog.AddButton to add the Buttons. More tedious, but what are you gonna do?
	//     That resulted in a crash, however.
	//     So the following trick, while looking very wrong, actually works:
	//     - Supply two buttons, Cancel and OK to DialogNewWithButtons
	//     - Add the OK button again with Dialog.AddButton
	//
	//     It's not a big drama, really, but it confused me quite a bit, so I thought I'd tell
	//     my story in case anyone ever walks into the same trap.
	//     I have no idea if this is a problem with Gtk, the Go bindings, or whatever, or maybe
	//     even expected behaviour.

	if dlg, err = gtk.DialogNewWithButtons(
		"Add Tag",
		g.win,
		gtk.DIALOG_MODAL,
		[]interface{}{
			"Cancel",
			gtk.RESPONSE_CANCEL,
			"OK",
			gtk.RESPONSE_OK,
		},
	); err != nil {
		g.log.Printf("Error creating gtk.Dialog: %s\n",
			err.Error())
		return
	}

	defer dlg.Close()

	if _, err = dlg.AddButton("OK", gtk.RESPONSE_OK); err != nil {
		g.log.Printf("[ERROR] Cannot add cancel button to AddTag Dialog: %s\n",
			err.Error())
		return
	} else if hbox, err = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 1); err != nil {
		g.log.Printf("[ERROR] Cannot create gtk.Box for AddTag Dialog: %s\n",
			err.Error())
		return
	} else if lbl, err = gtk.LabelNew("Name:"); err != nil {
		g.log.Printf("[ERROR] Cannot create Label for AddTag Dialog: %s\n",
			err.Error())
		return
	} else if entry, err = gtk.EntryNew(); err != nil {
		g.log.Printf("[ERROR] Cannot create Entry for AddTag Dialog: %s\n",
			err.Error())
		return
	} else if dbox, err = dlg.GetContentArea(); err != nil {
		g.log.Printf("[ERROR] Cannot get ContentArea of AddTag Dialog: %s\n",
			err.Error())
		return
	}

	dbox.PackStart(hbox, true, true, 0)
	hbox.PackStart(lbl, false, false, 0)
	hbox.PackStart(entry, true, true, 0)

	dlg.ShowAll()

	var (
		name string
		t    *objects.Tag
		res  = dlg.Run()
	)

	switch res {
	case gtk.RESPONSE_NONE:
		fallthrough
	case gtk.RESPONSE_DELETE_EVENT:
		fallthrough
	case gtk.RESPONSE_CLOSE:
		fallthrough
	case gtk.RESPONSE_CANCEL:
		g.log.Println("[DEBUG] User changed their mind about adding a Tag. Fine with me.")
		return
	case gtk.RESPONSE_OK:
		if name, err = entry.GetText(); err != nil {
			g.log.Printf("[ERROR] Cannot get Text from Dialog: %s\n",
				err.Error())
			return
		}

		g.log.Printf("[DEBUG] User wants to add a Tag named %q\n",
			name)
	default:
		g.log.Printf("[DEBUG] Well, I did NOT see this coming: %d\n",
			res)
	}

	if t, err = g.db.TagAdd(name); err != nil {
		var msg = fmt.Sprintf("Cannot add Tag %q to database: %s",
			name,
			err.Error())
		g.log.Printf("[ERROR] %s\n",
			msg)
		g.displayMsg(msg)
		return
	} else if err = g.tagAdd(t); err != nil {
		var msg = fmt.Sprintf("Cannot add Tag %s to UI: %s",
			t.Name,
			err.Error())
		g.log.Printf("[ERROR] %s\n",
			msg)
		g.displayMsg(msg)
	}
} // func (g *GUI) handleTagAdd()
