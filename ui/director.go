// /home/krylon/go/src/github.com/blicero/blockbuster/ui/director.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-21 22:56:49 krylon>

package ui

import (
	"fmt"

	"github.com/blicero/blockbuster/objects"
	"github.com/blicero/krylib"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func (g *GUI) removeDirector(p *objects.Person, f *objects.File) {
	krylib.Trace()
	defer g.log.Printf("[TRACE] EXIT %s\n",
		krylib.TraceInfo())
	var (
		err               error
		msg, title        string
		store             *gtk.TreeStore
		piter, fiter      *gtk.TreeIter
		ival              *glib.Value
		gval              interface{}
		id                int
		exists, found, ok bool
	)

	store = g.tabs[tiDirector].store.(*gtk.TreeStore)

	piter, exists = store.GetIterFirst()

	if !exists {
		// If there are no nodes at all for acting credits, that's weird,
		// but our work here is done in this case.
		return
	}

	for !found {
		if ival, err = store.GetValue(piter, 0); err != nil {
			msg = fmt.Sprintf("Cannot get Person ID from TreeIter: %s",
				err.Error())
			goto ERROR
		} else if gval, err = ival.GoValue(); err != nil {
			msg = fmt.Sprintf("Cannot get Go value from glib.Value: %s",
				err.Error())
			goto ERROR
		}

		if id = gval.(int); id == int(p.ID) {
			found = true
		} else if !store.IterNext(piter) {
			break
		}
	}

	if !found {
		// Once again, this should not happen, but if there is no node
		// for the given Person at all, there is nothing for us to do:
		return
	}

	// Now we need to check the Person node's children (which represents Files)
	// to find the one we are to remove.
	// Damn, I wish this was less tedious.
	if !store.IterNthChild(fiter, piter, 0) {
		// This should not happen either, a Person with no acting credits
		// should not be displayed in the the Director tab to begin with.
		// So in this case, we remove the Person node and are done.
		store.Remove(piter)
		return
	}

	found = false
	for !found {
		if ival, err = store.GetValue(fiter, 3); err != nil {
			msg = fmt.Sprintf("Cannot get File title from TreeIter: %s",
				err.Error())
			goto ERROR
		} else if gval, err = ival.GoValue(); err != nil {
			msg = fmt.Sprintf("Cannot get Go value from glib.Value: %s",
				err.Error())
			goto ERROR
		} else if title, ok = gval.(string); !ok {
			msg = fmt.Sprintf("Unexpected type for Column 'Films': %T (expected string)",
				gval)
			goto ERROR
		} else if title == f.DisplayTitle() {
			found = true
		} else if !store.IterNext(fiter) {
			break
		}
	}

	if !found {
		// Again, this shouldn't happen
		return
	}

	store.Remove(fiter)
	if store.IterNChildren(piter) == 0 {
		store.Remove(piter)
	}

ERROR:
	g.log.Printf("[ERROR] %s\n", msg)
	g.displayMsg(msg)
} // func (g *GUI) removeDirector(p *objects.Person, f *objects.File)
