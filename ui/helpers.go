// /home/krylon/go/src/github.com/blicero/blockbuster/ui/helpers.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-14 18:23:11 krylon>

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

func (g *GUI) displayMsg(msg string) {
	var (
		err error
		dlg *gtk.Dialog
		lbl *gtk.Label
		box *gtk.Box
	)

	if dlg, err = gtk.DialogNewWithButtons(
		"Message",
		g.win,
		gtk.DIALOG_MODAL,
		[]interface{}{
			"Okay",
			gtk.RESPONSE_OK,
		},
	); err != nil {
		g.log.Printf("[ERROR] Cannot create dialog to display message: %s\nMesage would've been %q\n",
			err.Error(),
			msg)
		return
	}

	defer dlg.Close()

	if lbl, err = gtk.LabelNew(msg); err != nil {
		g.log.Printf("[ERROR] Cannot create label to display message: %s\nMessage would've been: %q\n",
			err.Error(),
			msg)
		return
	} else if box, err = dlg.GetContentArea(); err != nil {
		g.log.Printf("[ERROR] Cannot get ContentArea of Dialog to display message: %s\nMessage would've been %q\n",
			err.Error(),
			msg)
		return
	}

	box.PackStart(lbl, true, true, 0)
	dlg.ShowAll()
	dlg.Run()
} // func (g *GUI) displayMsg(msg string)
