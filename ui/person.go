// /home/krylon/go/src/github.com/blicero/blockbuster/ui/person.go
// -*- mode: go; coding: utf-8; -*-
// Created on 14. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-14 04:35:16 krylon>

package ui

import (
	"fmt"

	"github.com/blicero/blockbuster/objects"
	"github.com/gotk3/gotk3/gtk"
)

func (g *GUI) loadPeople() bool {
	g.clearData(tiPerson)

	var (
		err    error
		msg    string
		pcnt   int
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

		store.SetValue(piter, 0, p.ID)
		store.SetValue(piter, 1, p.Name)
		store.SetValue(piter, 2, p.Birthday.Year())

		if files, err = g.db.ActorGetByPerson(p); err != nil {
			msg = fmt.Sprintf("Cannot load Files with acting credits by %s (%s): %s",
				p.Name,
				p.ID,
				err.Error())
			goto ERROR
		}

		for fidx := range files {
			var (
				fiter *gtk.TreeIter
				f     = &files[fidx]
			)

		}
	}

ERROR:
	g.log.Printf("[ERROR] %s\n", msg)
	g.displayMsg(msg)
	return false
} // func (g *GUI) loadPeople() bool
