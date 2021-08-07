// /home/krylon/go/src/github.com/blicero/blockbuster/tree/scanner.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-07 21:19:54 krylon>

// Package tree implements scanning directory trees for video files.
package tree

import (
	"log"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/blicero/blockbuster/common"
	"github.com/blicero/blockbuster/database"
	"github.com/blicero/blockbuster/logdomain"
	"github.com/blicero/blockbuster/objects"
)

const poolSize = 4

var suffixRe = regexp.MustCompile(suffixPattern)

// Scanner wraps all the handling of scanning Folders.
type Scanner struct {
	pool      *database.Pool
	log       *log.Logger
	lock      sync.RWMutex
	workerCnt int
	fileQ     chan<- *objects.File
}

// NewScanner creates a new Scanner that will handle the given list of paths.
// cnt is the number of goroutines to allocate for walking the directory trees
// in parallel.
func NewScanner(fileQ chan<- *objects.File) (*Scanner, error) {
	var (
		err error
		s   = &Scanner{
			fileQ: fileQ,
		}
	)

	if s.log, err = common.GetLogger(logdomain.Scanner); err != nil {
		return nil, err
	} else if s.pool, err = database.NewPool(poolSize); err != nil {
		s.log.Printf("[ERROR] Cannot open Database at %s: %s\n",
			common.DbPath,
			err.Error())
	}

	return s, nil
} // func NewScanner(cnt int) (*Scanner, error)

func (s *Scanner) addWorker() {
	s.lock.Lock()
	s.workerCnt++
	s.lock.Unlock()
} // func (s *Scanner) addWorker()

func (s *Scanner) delWorker() {
	s.lock.Lock()
	s.workerCnt--
	s.lock.Unlock()
} // func (s *Scanner) delWorker()

// Active returns true if the Scanner is currently scanning any Folders.
func (s *Scanner) Active() bool {
	s.lock.RLock()
	var active = s.workerCnt > 0
	s.lock.RUnlock()
	return active
} // func (s *Scanner) Active() bool

// ScanPath tells the Scanner to inspect the given directories.
// The scanning itself happens in separate goroutines (one per directory).
func (s *Scanner) ScanPath(paths ...string) {
	for _, path := range paths {
		s.log.Printf("[TRACE] Adding %q to scan queue\n",
			path)
		go s.scanFolder(path)
	}
} // func (s *Scanner) ScanPath(path string)

func (s *Scanner) scanFolder(path string) {
	var (
		err    error
		db     *database.Database
		folder *objects.Folder
	)

	s.addWorker()
	defer s.delWorker()

	db = s.pool.Get()
	defer s.pool.Put(db)

	if folder, err = db.FolderGetByPath(path); err != nil {
		s.log.Printf("[ERROR] Cannot look for Folder %q in Database: %s\n",
			path,
			err.Error())
		return
	} else if folder == nil {
		if folder, err = db.FolderAdd(path); err != nil {
			s.log.Printf("[ERROR] Cannot add Folder %s to database: %s\n",
				path,
				err.Error())
			return
		}
	}

	defer func() {
		var r error
		if r = db.FolderUpdateScan(folder, time.Now()); r != nil {
			s.log.Printf("[ERROR] Cannot update scan timestamp on Folder %q: %s\n",
				path,
				err.Error())
		}
	}()

	var w = walker{
		log:   s.log,
		root:  folder,
		fileQ: s.fileQ,
		db:    db,
	}

	if err = filepath.WalkDir(path, w.visitFile); err != nil {
		s.log.Printf("[ERROR] Failed to scan Folder %q: %s\n",
			path,
			err.Error())
	}
} // func (s *Scanner) scanFolder(path string)
