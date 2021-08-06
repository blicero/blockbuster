// /home/krylon/go/src/github.com/blicero/blockbuster/ui/helpers.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-05 20:44:39 krylon>

package ui

import "github.com/gotk3/gotk3/gtk"

func createCol(title string, id int) (*gtk.TreeViewColumn, error) {
	renderer, err := gtk.CellRendererTextNew()
	if err != nil {
		return nil, err
	}

	col, err := gtk.TreeViewColumnNewWithAttribute(title, renderer, "text", id)
	if err != nil {
		return nil, err
	}

	return col, nil
} // func createCol(title string, id int) (*gtk.TreeViewColumn, error)
