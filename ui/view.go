// /home/krylon/go/src/github.com/blicero/blockbuster/ui/view.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-21 23:17:14 krylon>

// The GUI makes generous use of Gtk's TreeView.
// While TreeView is very versatile and awesome, it can also be very tedious to
// deal with. In order to make this less tedious, annoying and error-prone, I try
// to automate away as much of the tedium as possible.

package ui

import (
	"fmt"

	"github.com/blicero/krylib"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type tabIdx uint8

// nolint: deadcode,unused,varcheck
const (
	tiFile tabIdx = iota
	tiActor
	tiDirector
	tiTags
	tiPerson
	tiFolder
)

type storeType uint8

const (
	storeList storeType = iota
	storeTree
)

type column struct {
	colType glib.Type
	title   string
	edit    bool
}

type cellEditHandlerFactory func(int) func(*gtk.CellRendererText, string, string)

type view struct {
	title   string
	store   storeType
	columns []column
}

func (v *view) typeList() []glib.Type {
	krylib.Trace()
	defer krylib.Trace()
	var res = make([]glib.Type, len(v.columns))

	for i, c := range v.columns {
		res[i] = c.colType
	}

	return res
} // func (v *view) typeList() []glib.Type

func (v *view) create(handlerFactory cellEditHandlerFactory) (gtk.ITreeModel, *gtk.TreeView, error) {
	krylib.Trace()
	defer krylib.Trace()
	var (
		err   error
		cols  []glib.Type
		store gtk.ITreeModel
		tv    *gtk.TreeView
	)

	cols = v.typeList()
	switch v.store {
	case storeList:
		if store, err = gtk.ListStoreNew(cols...); err != nil {
			return nil, nil, err
		}
	case storeTree:
		if store, err = gtk.TreeStoreNew(cols...); err != nil {
			return nil, nil, err
		}
	default:
		err = fmt.Errorf("invalid Store type %d", v.store)
		return nil, nil, err
	}

	if tv, err = gtk.TreeViewNewWithModel(store); err != nil {
		return nil, nil, err
	}

	for idx, cSpec := range v.columns {
		var (
			col      *gtk.TreeViewColumn
			renderer *gtk.CellRendererText
		)
		if col, renderer, err = createCol(cSpec.title, idx); err != nil {
			return nil, nil, err
		}

		renderer.Set("editable", cSpec.edit)     // nolint: errcheck
		renderer.Set("editable-set", cSpec.edit) // nolint: errcheck
		if cSpec.edit && handlerFactory != nil {
			renderer.Connect("edited", handlerFactory(idx))
		}

		tv.AppendColumn(col)
	}

	return store, tv, nil
} // func (v *view) create(handlerFactory cellEditHandlerFactory) (gtk.ITreeModel, *gtk.TreeView, error)

var viewList = []view{
	view{
		title: "File",
		store: storeList,
		columns: []column{
			column{
				colType: glib.TYPE_INT,
				title:   "ID",
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Title",
				edit:    true,
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Size",
			},
			column{
				colType: glib.TYPE_INT,
				title:   "Year",
				edit:    true,
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Director",
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Actor(s)",
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Tags",
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Path",
			},
		},
	},
	view{
		title: "Actor",
		store: storeTree,
		columns: []column{
			column{
				colType: glib.TYPE_INT,
				title:   "ID",
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Name",
			},
			column{
				colType: glib.TYPE_INT,
				title:   "Born",
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Films",
			},
		},
	},
	view{
		title: "Director",
		store: storeTree,
		columns: []column{
			column{
				colType: glib.TYPE_INT,
				title:   "ID",
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Name",
			},
			column{
				colType: glib.TYPE_INT,
				title:   "Born",
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Films",
			},
		},
	},
	view{
		title: "Tags",
		store: storeTree,
		columns: []column{
			column{
				colType: glib.TYPE_INT,
				title:   "ID",
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Name",
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Films",
			},
			column{
				colType: glib.TYPE_INT,
				title:   "Year",
			},
		},
	},
	view{
		title: "Person",
		store: storeTree,
		columns: []column{
			column{
				colType: glib.TYPE_INT,
				title:   "ID",
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Name",
			},
			column{
				colType: glib.TYPE_INT,
				title:   "Birthday",
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Films",
			},
		},
	},
	view{
		title: "Folder",
		store: storeList,
		columns: []column{
			column{
				colType: glib.TYPE_INT,
				title:   "ID",
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Path",
			},
			column{
				colType: glib.TYPE_STRING,
				title:   "Last Scan",
			},
		},
	},
}
