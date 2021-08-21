// /home/krylon/go/src/github.com/blicero/blockbuster/ui/ui.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-21 21:01:07 krylon>

// Package ui provides the user interface for the video library.
package ui

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/anmitsu/go-shlex"
	"github.com/blicero/blockbuster/common"
	"github.com/blicero/blockbuster/database"
	"github.com/blicero/blockbuster/logdomain"
	"github.com/blicero/blockbuster/objects"
	"github.com/blicero/blockbuster/tree"
	"github.com/blicero/krylib"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

const (
	statusPlayer   uint = iota
	statusBeacon        // nolint: deadcode,unused,varcheck
	statusSearch        // nolint: deadcode,unused,varcheck
	statusScan          // nolint: deadcode,unused,varcheck
	statusInternet      // nolint: deadcode,unused,varcheck
)

const (
	qDepth        = 128
	refInterval   = time.Second * 10
	defaultPlayer = "/usr/bin/mpv"
	playerEnv     = "VIDEOPLAYER"
)

type tabContent struct {
	vbox   *gtk.Box
	sbox   *gtk.Box
	lbl    *gtk.Label
	search *gtk.Entry
	store  gtk.ITreeModel
	view   *gtk.TreeView
	scr    *gtk.ScrolledWindow
}

// GUI is the ... well, GUI of the application.
// Built using Gtk3, which I am still in the process of learning,
// with the added challenge of using a C library from Go, with the
// added challenge that I basically suck at UI design.
//
// So, you've been warned, if you stick around, some interesting times
// lie ahead!
type GUI struct {
	db        *database.Database
	scanner   *tree.Scanner
	log       *log.Logger
	fileQ     chan *objects.File
	lock      sync.RWMutex // nolint: structcheck,unused
	win       *gtk.Window
	mainBox   *gtk.Box
	menubar   *gtk.MenuBar
	notebook  *gtk.Notebook
	statusbar *gtk.Statusbar
	tabs      []tabContent
	tags      objects.TagList
	playCmd   []string
}

// Create creates a new GUI. You didn't see *that* coming, now, did you?
func Create() (*GUI, error) {
	var (
		err          error
		playerString string
		g            = &GUI{
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

	if playerString = os.Getenv(playerEnv); playerString == "" {
		playerString = defaultPlayer
	}

	if g.playCmd, err = shlex.Split(playerString, true); err != nil {
		g.log.Printf("[ERROR] Cannot parse player command: %s\n",
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
	} else if g.statusbar, err = gtk.StatusbarNew(); err != nil {
		g.log.Printf("[ERROR] Cannot create Statusbar: %s\n",
			err.Error())
		return nil, err
	}

	if err = g.initMenu(); err != nil {
		g.log.Printf("[ERROR] Failed to create menu: %s\n",
			err.Error())
		return nil, err
	}

	g.tabs = make([]tabContent, len(viewList))

	for tIdx, v := range viewList {
		var (
			tab         tabContent
			lbl         *gtk.Label
			editHandler cellEditHandlerFactory
		)

		switch tabIdx(tIdx) {
		case tiFile:
			editHandler = g.mkFileEditHandler
		}

		if tab.store, tab.view, err = v.create(editHandler); err != nil {
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
		} else if tab.vbox, err = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 1); err != nil {
			g.log.Printf("[ERROR] Cannot create vbox for %q: %s\n",
				v.title,
				err.Error())
			return nil, err
		} else if tab.sbox, err = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 1); err != nil {
			g.log.Printf("[ERROR] Cannot create sbox for %q: %s\n",
				v.title,
				err.Error())
			return nil, err
		} else if tab.lbl, err = gtk.LabelNew("Filter:"); err != nil {
			g.log.Printf("[ERROR] Cannot create Filter Label for %s: %s\n",
				v.title,
				err.Error())
			return nil, err
		} else if tab.search, err = gtk.EntryNew(); err != nil {
			g.log.Printf("[ERROR] Cannot create search entry for %s: %s\n",
				v.title,
				err.Error())
			return nil, err
		}

		tab.sbox.PackStart(tab.lbl, false, false, 1)
		tab.sbox.PackStart(tab.search, true, true, 1)
		tab.vbox.PackStart(tab.sbox, false, false, 1)
		tab.vbox.PackStart(tab.scr, true, true, 1)

		g.tabs[tIdx] = tab
		tab.scr.Add(tab.view)
		g.notebook.AppendPage(tab.vbox, lbl)

	}

	// Edit Handlers

	////////////////////////////////////////////////////////////////////////////////
	///// Context Menus ////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////////////

	// One thing I really liked about the old Ruby application was that I
	// had sensible context menus, so I could right-click any object in a
	// tree view and get a meaningful list of things to do.

	g.tabs[tiFile].view.Connect("button-press-event", g.handleFileListClick)
	g.tabs[tiPerson].view.Connect("button-press-event", g.handlePersonListClick)
	// g.tabs[tiFolder].view.Connect("button-press-event", g.handleFileListClick)

	g.win.Connect("destroy", gtk.MainQuit)

	g.mainBox.PackStart(g.menubar, false, false, 0)
	g.mainBox.PackStart(g.notebook, true, true, 0)
	g.mainBox.PackStart(g.statusbar, false, false, 0)
	g.win.Add(g.mainBox)
	g.win.SetSizeRequest(960, 540)
	g.win.SetTitle(fmt.Sprintf("%s %s",
		common.AppName,
		common.Version))

	return g, nil
} // func Create() (*GUI, error)

// ShowAndRun displays the GUI and runs the Gtk event loop.
func (g *GUI) ShowAndRun() {
	krylib.Trace()
	if err := g.loadData(); err != nil {
		g.log.Printf("[ERROR] Cannot load data: %s\n",
			err.Error())
		return
	}

	go g.scanLoop()

	g.win.ShowAll()
	// glib.TimeoutAdd(1000, g.beacon)
	gtk.Main()
} // func (g *GUI) ShowAndRun()

// func (g *GUI) beacon() bool {
// 	krylib.Trace()
// 	var msg = fmt.Sprintf("Beacon is alive at %s",
// 		time.Now().Format(common.TimestampFormatTime))
// 	g.log.Printf("[TRACE] %s\n", msg)
// 	g.statusbar.Push(statusBeacon, msg)
// 	return true
// }

func (g *GUI) scanLoop() {
	krylib.Trace()
	defer func() {
		krylib.Trace()
		g.log.Println("[INFO] GUI Scanner loop is quitting. So long, suckers!")
	}()

	var ticker = time.NewTicker(refInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			g.log.Println("[TRACE] scanLoop says Hello.")
			continue
		case f := <-g.fileQ:
			g.log.Printf("[DEBUG] Received new File %d: %s\n",
				f.ID,
				f.Path)
			glib.IdleAdd(g.makeNewFileHandler(f))
		}
	}
} // func (g *GUI) scanLoop()

func (g *GUI) loadData() error {
	var (
		err        error
		fileList   []objects.File
		folderList []objects.Folder
		actorList  []objects.Person
	)

	krylib.Trace()

	if fileList, err = g.db.FileGetAll(); err != nil {
		g.log.Printf("[ERROR] Cannot get list of all Files: %s\n",
			err.Error())
		return err
	}

	for idx := range fileList {
		var handler = g.makeNewFileHandler(&fileList[idx])
		glib.IdleAdd(handler)
	}

	if folderList, err = g.db.FolderGetAll(); err != nil {
		g.log.Printf("[ERROR] Cannot get list of all Folders: %s\n",
			err.Error())
		return err
	}

	for idx := range folderList {
		var handler = g.makeNewFolderHandler(&folderList[idx])
		glib.IdleAdd(handler)
	}

	if actorList, err = g.db.PersonGetAll(); err != nil {
		g.log.Printf("[ERROR] Cannot get list of all Persons: %s\n",
			err.Error())
		return err
	}

	for pidx := range actorList {
		var p = &actorList[pidx]
		if fileList, err = g.db.ActorGetByPerson(p); err != nil {
			g.log.Printf("[ERROR] Cannot get list of acting credits for %s: %s\n",
				p.Name,
				err.Error())
			return err
		} else if len(fileList) == 0 {
			continue
		}

		for fidx := range fileList {
			var f = &fileList[fidx]
			var handler = g.makeNewActorHandler(p, f)
			glib.IdleAdd(handler)
		}
	}

	g.loadTagView()
	g.loadPeople()

	return nil
} // func (g *GUI) loadData() error

func (g *GUI) clearData(idx tabIdx) {
	krylib.Trace()
	switch s := g.tabs[idx].store.(type) {
	case *gtk.ListStore:
		s.Clear()
	case *gtk.TreeStore:
		s.Clear()
	default:
		var msg = fmt.Sprintf("Unexpected type for TreeModel: %T",
			g.tabs[idx].store)
		g.log.Printf("[CANTHAPPEN] %s\n", msg)
		g.displayMsg(msg)
		return
	}
} // func (g *GUI) clearData(idx tabIdx)

func (g *GUI) reloadData() {
	krylib.Trace()
	for idx := range g.tabs {
		g.clearData(tabIdx(idx))
	}

	if err := g.loadData(); err != nil {
		var msg = fmt.Sprintf("Failed to load data: %s",
			err.Error())
		g.log.Printf("[ERROR] %s\n", msg)
		g.displayMsg(msg)
	}
} // func (g *GUI) reloadData()

func (g *GUI) makeNewFileHandler(f *objects.File) func() bool {
	krylib.Trace()
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
		krylib.Trace()
		var (
			err                 error
			astr, tstr, sizeStr string
			size                int64
			iter                = store.Append()
		)

		if f.ID != 0 {
			var (
				actors []objects.Person
				tags   map[int64]objects.Tag
				slist  []string
			)

			if tags, err = g.db.TagLinkGetByFile(f); err != nil {
				g.log.Printf("[ERROR] Cannot get Tags for File %s: %s\n",
					f.DisplayTitle(),
					err.Error())
			} else {
				slist = make([]string, 0, len(tags))
				for _, t := range tags {
					slist = append(slist, t.Name)
				}

				sort.Strings(slist)
				tstr = strings.Join(slist, ", ")
			}

			if actors, err = g.db.ActorGetByFile(f); err != nil {
				g.log.Printf("[ERROR] Cannot get Actors for File %s: %s\n",
					f.DisplayTitle(),
					err.Error())
			} else {
				slist = make([]string, len(actors))
				for i, p := range actors {
					slist[i] = p.Name
				}
				astr = strings.Join(slist, ", ")
			}
		}

		if size = f.Size(); size != 0 {
			sizeStr = krylib.FmtBytes(size)
		}

		if err = store.Set(
			iter,
			[]int{0, 1, 2, 3, 5, 6, 7},
			[]interface{}{f.ID, f.DisplayTitle(), sizeStr, f.Year, astr, tstr, f.Path},
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
	krylib.Trace()
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
		krylib.Trace()
		var (
			err  error
			iter = store.Append()
		)

		if err = store.Set(
			iter,
			[]int{0, 1, 2},
			[]interface{}{f.ID, f.Path, f.LastScan.Format(common.TimestampFormat)},
		); err != nil {
			g.log.Printf("[ERROR] Cannot add FOlder %d (%s) to Store: %s\n",
				f.ID,
				f.Path,
				err.Error())
		}

		return false
	}
} // func (g *GUI) makeNewFolderHandler(f *objects.Folder) func() bool

func (g *GUI) makeNewActorHandler(p *objects.Person, f *objects.File) func() bool {
	krylib.Trace()
	return func() bool {
		krylib.Trace()
		var (
			err         error
			msg         string
			iter, fiter *gtk.TreeIter
			exists      bool
			pos         int
			store       = g.tabs[tiActor].store.(*gtk.TreeStore)
		)

		iter, exists = store.GetIterFirst()

		if !exists {
			iter = store.Append(nil)
			store.SetValue(iter, 0, p.ID)              // nolint: errcheck
			store.SetValue(iter, 1, p.Name)            // nolint: errcheck
			store.SetValue(iter, 2, p.Birthday.Year()) // nolint: errcheck
		} else {
			var (
				ival  *glib.Value
				gval  interface{}
				found bool
			)

			for !found {
				if ival, err = store.GetValue(iter, 0); err != nil {
					msg = fmt.Sprintf("Cannot get Person ID from TreeIter: %s",
						err.Error())
					goto ERROR
				} else if gval, err = ival.GoValue(); err != nil {
					msg = fmt.Sprintf("Cannot get Go value from glib.Value: %s",
						err.Error())
					goto ERROR
				}

				var id = gval.(int)
				if id == int(p.ID) {
					found = true
				} else if !store.IterNext(iter) {
					break
				}
			}

			if !found {
				iter = store.Append(nil)
				store.SetValue(iter, 0, p.ID)              // nolint: errcheck
				store.SetValue(iter, 1, p.Name)            // nolint: errcheck
				store.SetValue(iter, 2, p.Birthday.Year()) // nolint: errcheck
			}
		}

		// iter now points to the node of the Person
		pos = store.IterNChildren(iter)
		fiter = store.Insert(iter, pos+1)
		store.SetValue(fiter, 3, f.DisplayTitle()) // nolint: errcheck

		return false
	ERROR:
		g.log.Printf("[ERROR] %s\n", msg)
		g.displayMsg(msg)

		return false
	}
} // func (g *GUI) makeNewActorHandler(p *objects.Person, f *objects.File) func ()

func (g *GUI) makeNewDirectorHandler(p *objects.Person, f *objects.File) func() bool {
	krylib.Trace()
	return func() bool {
		krylib.Trace()
		var (
			err         error
			msg         string
			iter, fiter *gtk.TreeIter
			exists      bool
			pos         int
			store       = g.tabs[tiDirector].store.(*gtk.TreeStore)
		)

		iter, exists = store.GetIterFirst()

		if !exists {
			iter = store.Append(nil)
			store.SetValue(iter, 0, p.ID)              // nolint: errcheck
			store.SetValue(iter, 1, p.Name)            // nolint: errcheck
			store.SetValue(iter, 2, p.Birthday.Year()) // nolint: errcheck
		} else {
			var (
				ival  *glib.Value
				gval  interface{}
				found bool
			)

			for !found {
				if ival, err = store.GetValue(iter, 0); err != nil {
					msg = fmt.Sprintf("Cannot get Person ID from TreeIter: %s",
						err.Error())
					goto ERROR
				} else if gval, err = ival.GoValue(); err != nil {
					msg = fmt.Sprintf("Cannot get Go value from glib.Value: %s",
						err.Error())
					goto ERROR
				}

				var id = gval.(int)
				if id == int(p.ID) {
					found = true
				} else if !store.IterNext(iter) {
					break
				}
			}

			if !found {
				iter = store.Append(nil)
				store.SetValue(iter, 0, p.ID)              // nolint: errcheck
				store.SetValue(iter, 1, p.Name)            // nolint: errcheck
				store.SetValue(iter, 2, p.Birthday.Year()) // nolint: errcheck
			}
		}

		// iter now points to the node of the Person
		pos = store.IterNChildren(iter)
		fiter = store.Insert(iter, pos+1)
		store.SetValue(fiter, 3, f.DisplayTitle()) // nolint: errcheck

		return false
	ERROR:
		g.log.Printf("[ERROR] %s\n", msg)
		g.displayMsg(msg)

		return false
	}
} // func (g *GUI) makeNewDirectorHandler(p *objects.Person, f *objects.File) func ()

func (g *GUI) promptScanFolder() {
	krylib.Trace()
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

func (g *GUI) handleTagAdd() {
	krylib.Trace()
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

func (g *GUI) handlePersonAdd() {
	krylib.Trace()
	var (
		err              error
		dlg              *gtk.Dialog
		dbox             *gtk.Box
		grid             *gtk.Grid
		nameLbl, bdayLbl *gtk.Label
		entry            *gtk.Entry
		cal              *gtk.Calendar
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
		"Add Person",
		g.win,
		gtk.DIALOG_MODAL,
		[]interface{}{
			"Cancel",
			gtk.RESPONSE_CANCEL,
			"OK",
			gtk.RESPONSE_OK,
		},
	); err != nil {
		g.log.Printf("[ERROR] Error creating gtk.Dialog: %s\n",
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
	} else if nameLbl, err = gtk.LabelNew("Name:"); err != nil {

		g.log.Printf("[ERROR] Cannot create name Label for AddPerson Dialog: %s\n",
			err.Error())
		return
	} else if bdayLbl, err = gtk.LabelNew("Birthday:"); err != nil {
		g.log.Printf("[ERROR] Cannot create birthday Label for AddPerson Dialog: %s\n",
			err.Error())
		return
	} else if entry, err = gtk.EntryNew(); err != nil {
		g.log.Printf("[ERROR] Cannot create Entry for AddPerson Dialog: %s\n",
			err.Error())
		return
	} else if cal, err = gtk.CalendarNew(); err != nil {
		g.log.Printf("[ERROR] Cannot create Calendar for AddPerson Dialog: %s\n",
			err.Error())
	} else if dbox, err = dlg.GetContentArea(); err != nil {
		g.log.Printf("[ERROR] Cannot get ContentArea of AddPerson Dialog: %s\n",
			err.Error())
		return
	}

	grid.InsertColumn(0)
	grid.InsertColumn(1)
	grid.InsertRow(0)
	grid.InsertRow(1)

	grid.Attach(nameLbl, 0, 0, 1, 1)
	grid.Attach(bdayLbl, 0, 1, 1, 1)
	grid.Attach(entry, 1, 0, 1, 1)
	grid.Attach(cal, 1, 1, 1, 1)

	dbox.PackStart(grid, true, true, 0)

	dlg.ShowAll()

	var (
		name    string
		y, m, d uint
		bday    time.Time
		person  *objects.Person
		res     = dlg.Run()
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

		y, m, d = cal.GetDate()
		bday = krylib.Date(int(y), int(m), int(d))

		g.log.Printf("[DEBUG] User wants to add a Person named %q\n",
			name)
	default:
		g.log.Printf("[DEBUG] Well, I did NOT see this coming: %d\n",
			res)
		return
	}

	if person, err = g.db.PersonAdd(name, bday); err != nil {
		var msg = fmt.Sprintf("Cannot add Person %q to database: %s",
			name,
			err.Error())
		g.log.Printf("[ERROR] %s\n",
			msg)
		g.displayMsg(msg)
		return
	} /*else if err = g.tagAdd(t); err != nil {
		var msg = fmt.Sprintf("Cannot add Tag %s to UI: %s",
			person.Name,
			err.Error())
		g.log.Printf("[ERROR] %s\n",
			msg)
		g.displayMsg(msg)
	} */

	g.log.Printf("[DEBUG] Person %s (born %s) was added to Database\n",
		person.Name,
		person.Birthday.Format(common.TimestampFormatDate))
} // func (g *GUI) handlerPersonAdd()

func (g *GUI) playFile(f *objects.File) {
	krylib.Trace()
	var (
		err  error
		cmd  *exec.Cmd
		args = make([]string, len(g.playCmd))
	)

	for i, a := range g.playCmd[1:] {
		args[i] = a
	}
	args[len(args)-1] = f.Path

	cmd = exec.Command(g.playCmd[0], args...)

	if err = cmd.Start(); err != nil {
		var msg = fmt.Sprintf("Failed to start player for %s: %s",
			f.DisplayTitle(),
			err.Error())
		g.log.Printf("[ERROR] %s\n", msg)
		g.displayMsg(msg)
	}

	var msg = fmt.Sprintf("Playing %s", f.DisplayTitle())
	g.statusbar.Push(statusPlayer, msg)

	go func() {
		krylib.Trace()
		var e error
		if e = cmd.Wait(); e != nil {
			var m = fmt.Sprintf("Error playing %q: %s",
				f.DisplayTitle(),
				e.Error())
			g.log.Printf("[ERROR] %s\n", m)
			glib.IdleAdd(func() bool {
				krylib.Trace()
				g.statusbar.Push(statusPlayer, m)
				return false
			})
		} else {
			g.log.Printf("[TRACE] Playing %s finished.\n",
				f.DisplayTitle())
			glib.IdleAdd(func() bool {
				krylib.Trace()
				var m = fmt.Sprintf("Finished playing %s",
					f.DisplayTitle())
				g.statusbar.Push(statusPlayer, m)
				return false
			})
		}
	}()
} // func (g *GUI) playFile(f *objects.File)
