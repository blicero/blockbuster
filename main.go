// /home/krylon/go/src/github.com/blicero/blockbuster/main.go
// -*- mode: go; coding: utf-8; -*-
// Created on 06. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-06 12:05:25 krylon>

package main

import (
	"fmt"
	"os"

	"github.com/blicero/blockbuster/common"
	"github.com/blicero/blockbuster/ui"
)

func main() {
	var (
		err error
		win *ui.GUI
	)

	if err = common.InitApp(); err != nil {
		fmt.Fprintf(os.Stderr,
			"Cannot initialize application environment: %s\n",
			err.Error())
		os.Exit(1)
	} else if win, err = ui.Create(); err != nil {
		fmt.Fprintf(os.Stderr,
			"Cannot create GUI: %s\n",
			err.Error())
		os.Exit(1)
	}

	win.ShowAndRun()
} // func main()
