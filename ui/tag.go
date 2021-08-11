// /home/krylon/go/src/github.com/blicero/blockbuster/ui/tag.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-11 17:14:12 krylon>

package ui

import (
	"sort"

	"github.com/blicero/blockbuster/objects"
	"github.com/gotk3/gotk3/gtk"
)

func (g *GUI) tagAdd(t *objects.Tag) error {
	var (
		err   error
		model *gtk.TreeStore
		iter  *gtk.TreeIter
	)

	model = g.tabs[tiTags].store.(*gtk.TreeStore)

	iter = model.Append(nil)

	if err = model.SetValue(iter, 0, t.ID); err != nil {
		g.log.Printf("[ERROR] Cannot set ID for Tag %s: %s\n",
			t.Name,
			err.Error())
		return err
	} else if err = model.SetValue(iter, 1, t.Name); err != nil {
		g.log.Printf("[ERROR] Cannot set Name for Tag %s: %s\n",
			t.Name,
			err.Error())
		return err
	}

	g.tags = append(g.tags, *t)
	sort.Sort(g.tags)

	return nil
} // func (g *GUI) tagAdd(t *objects.Tag) error
