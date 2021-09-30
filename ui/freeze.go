// /home/krylon/go/src/github.com/blicero/blockbuster/freeze.go
// -*- mode: go; coding: utf-8; -*-
// Created on 23. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-26 19:10:25 krylon>

package ui

import (
	"os"
	"time"
)

const heartbeatTimeout = time.Millisecond * 2500

type heartbeatCounter int

var (
	aliveCnt   heartbeatCounter = 0
	heartbeatQ                  = make(chan heartbeatCounter, 2)
)

func (g *GUI) heartbeat() bool {
	aliveCnt++
	heartbeatQ <- aliveCnt
	return true
} // func (g *GUI) heartbeat()

func (g *GUI) heartbeatLoop() {
	const maxMiss = 8
	var (
		timeout = time.NewTicker(heartbeatTimeout)
		cnt     heartbeatCounter
		missCnt = 0
	)

	defer timeout.Stop()

	for {
		select {
		case cnt = <-heartbeatQ:
			missCnt = 0
		case <-timeout.C:
			missCnt++
			if missCnt > maxMiss {
				g.log.Printf("[CRITICAL] It would seem the GUI has frozen after %d heartbeats: %d missed heartbeats\n",
					cnt,
					missCnt)
				os.Exit(1)
			} else if missCnt > 1 {
				g.log.Printf("[CRITICAL] Gtk3 Main loop has missed %d/%d heartbeats in a row\n",
					missCnt,
					maxMiss)
			}
		}
	}
} // func heartbeatLoop()
