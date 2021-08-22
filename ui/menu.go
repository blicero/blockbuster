// /home/krylon/go/src/github.com/blicero/blockbuster/ui/menu.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-21 23:09:09 krylon>

package ui

import (
	"github.com/blicero/krylib"
	"github.com/gotk3/gotk3/gtk"
)

// Creating the menu is so tedious and verbose I am putting that part into a
// separate file.

func (g *GUI) initMenu() error {
	krylib.Trace()
	defer g.log.Printf("[TRACE] EXIT %s\n",
		krylib.TraceInfo())
	///////////////////////////////////////////////////////////////////////
	////// Menus //////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////////////////

	// FIXME I think I should somehow make the menu more data driven. Kinda like the TreeViews.

	var (
		err                                    error
		fileMenu, addMenu                      *gtk.Menu
		scanItem, reloadItem, quitItem, fmItem *gtk.MenuItem
		itemAddTag, itemAddPerson, amItem      *gtk.MenuItem
	)

	if fileMenu, err = gtk.MenuNew(); err != nil {
		g.log.Printf("[ERROR] Cannot create File menu: %s\n",
			err.Error())
		return err
	} else if scanItem, err = gtk.MenuItemNewWithMnemonic("_Scan"); err != nil {
		g.log.Printf("[ERROR] Cannot create menu item File/Scan: %s\n",
			err.Error())
		return err
	} else if reloadItem, err = gtk.MenuItemNewWithMnemonic("_Reload"); err != nil {
		g.log.Printf("[ERROR] Cannot create menu item File/Reload: %s\n",
			err.Error())
		return err
	} else if quitItem, err = gtk.MenuItemNewWithMnemonic("_Quit"); err != nil {
		g.log.Printf("[ERROR] Cannot create menu item File/Quit: %s\n",
			err.Error())
		return err
	} else if fmItem, err = gtk.MenuItemNewWithMnemonic("_File"); err != nil {
		g.log.Printf("[ERROR] Cannot create menu item File/: %s\n",
			err.Error())
		return err
	}

	if addMenu, err = gtk.MenuNew(); err != nil {
		g.log.Printf("[ERROR] Cannot create Add menu: %s\n",
			err.Error())
		return err
	} else if itemAddTag, err = gtk.MenuItemNewWithMnemonic("_Tag"); err != nil {
		g.log.Printf("[ERROR] Cannot create menu item Add/Tag: %s\n",
			err.Error())
		return err
	} else if itemAddPerson, err = gtk.MenuItemNewWithMnemonic("_Person"); err != nil {
		g.log.Printf("[ERROR] Cannot create menu item Add/Person: %s\n",
			err.Error())
		return err
	} else if amItem, err = gtk.MenuItemNewWithMnemonic("_Add"); err != nil {
		g.log.Printf("[ERROR] Cannot create menu Item Add/: %s\n",
			err.Error())
		return err
	}

	scanItem.Connect("activate", g.promptScanFolder)
	reloadItem.Connect("activate", g.reloadData)
	quitItem.Connect("activate", gtk.MainQuit)

	fmItem.SetSubmenu(fileMenu)

	fileMenu.Append(scanItem)
	fileMenu.Append(reloadItem)
	fileMenu.Append(quitItem)

	g.menubar.Append(fmItem)

	amItem.SetSubmenu(addMenu)
	addMenu.Append(itemAddTag)
	addMenu.Append(itemAddPerson)

	itemAddTag.Connect("activate", g.handleTagAdd)
	itemAddPerson.Connect("activate", g.handlePersonAdd)

	g.menubar.Append(amItem)

	return nil
} // func (g *GUI) initMenu() error
