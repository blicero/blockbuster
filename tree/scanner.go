// /home/krylon/go/src/github.com/blicero/blockbuster/tree/scanner.go
// -*- mode: go; coding: utf-8; -*-
// Created on 05. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-05 22:13:40 krylon>

// Package tree implements scanning directory trees for video files.
package tree

import (
	"io/fs"
	"log"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/blicero/blockbuster/common"
	"github.com/blicero/blockbuster/database"
	"github.com/blicero/blockbuster/logdomain"
	"github.com/blicero/blockbuster/objects"
	"github.com/blicero/krylib"
)

const (
	minSize       = 1024 * 1024 * 32 // 32 MB, minimum size for files to consider
	qSize         = 256              // ??? How do I determine what is a good size?
	suffixPattern = "(?i)[.](?:avi|mp4|mpg|asf|avi|flv|m4v|mkv|mov|mpg|ogm|ogv|sfv|webm|wmv)$"
	timeout       = time.Second
)

var suffixRe = regexp.MustCompile(suffixPattern)

// XXX Do I even need a database instance for the Scanner as a whole?

// Scanner wraps all the handling of scanning Folders.
type Scanner struct {
	db        *database.Database
	log       *log.Logger
	active    bool
	lock      sync.RWMutex
	fileQ     chan string
	scanQ     chan string
	workerCnt int
}

// NewScanner creates a new Scanner that will handle the given list of paths.
// cnt is the number of goroutines to allocate for walking the directory trees
// in parallel.
func NewScanner(cnt int) (*Scanner, error) {
	var (
		err error
		s   = &Scanner{
			fileQ: make(chan string, qSize),
			scanQ: make(chan string, qSize),
		}
	)

	if cnt <= 0 {
		s.workerCnt = runtime.NumCPU()
	} else {
		s.workerCnt = cnt
	}

	if s.log, err = common.GetLogger(logdomain.Scanner); err != nil {
		return nil, err
	} else if s.db, err = database.Open(common.DbPath); err != nil {
		s.log.Printf("[ERROR] Cannot open Database at %s: %s\n",
			common.DbPath,
			err.Error())
	}

	return s, nil
} // func NewScanner(cnt int) (*Scanner, error)

// Active returns true if the Scanner is currently active.
func (s *Scanner) Active() bool {
	s.lock.RLock()
	var active = s.active
	s.lock.RUnlock()
	return active
} // func (s *Scanner) Active() bool

// Start starts the Scanner if it is not already active.
func (s *Scanner) Start(newFileQ chan<- *objects.File) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.active {
		s.log.Println("[INFO] Scanner is already active.")
		return nil
	}

	s.log.Printf("[INFO] Starting Scanner with %d workers.\n",
		s.workerCnt)

	var (
		err error
		db  *database.Database
	)

	if db, err = database.Open(common.DbPath); err != nil {
		s.log.Printf("[ERROR] Cannot open Database: %s\n",
			err.Error())
		return err
	}

	s.active = true

	go s.gatherFiles(db, newFileQ)

	for i := 0; i < s.workerCnt; i++ {
		go s.worker(i + 1)
	}

	return nil
} // func (s *Scanner) Start()

// ScanPath adds the given path to the scan queue.
// The actual scanning happens in a separate goroutine.
func (s *Scanner) ScanPath(paths ...string) {
	for _, path := range paths {
		s.scanQ <- path
	}
} // func (s *Scanner) ScanPath(path string)

// Stop tells the Scanner to stop if it is currently active.
func (s *Scanner) Stop() {
	s.lock.Lock()
	s.log.Println("[INFO] Stopping Scanner")
	s.active = false
	s.lock.Unlock()
} // func (s *Scanner) Stop()

func (s *Scanner) worker(id int) {
	s.log.Printf("[INFO] Starting Scanner worker %02d.\n",
		id)

	defer s.log.Printf("[INFO] Scanner worker %02d is quitting.\n",
		id)

	var ticker = time.NewTicker(timeout)
	defer ticker.Stop()

	for s.Active() {
		select {
		case path := <-s.scanQ:
			s.log.Printf("[INFO] Scan path %s.\n",
				path)
			s.scanFolder(path)
		case <-ticker.C:
			continue
		}
	}
} // func (s *Scanner) worker(id int)

func (s *Scanner) scanFolder(path string) {
	var err error

	if err = filepath.WalkDir(path, s.visitFileFunc); err != nil {
		s.log.Printf("[ERROR] Error walking %s: %s\n",
			path,
			err.Error())
	}
} // func (s *Scanner) scanFolder(path string)

func (s *Scanner) visitFileFunc(path string, d fs.DirEntry, e error) error {
	if e != nil {
		s.log.Printf("[ERROR] Incoming error when visiting %s: %s\n",
			path,
			e.Error())
		return fs.SkipDir
	} else if !suffixRe.MatchString(path) {
		s.log.Printf("[TRACE] Skip %q -- suffix\n", path)
		return nil
	} else if !d.Type().IsRegular() {
		s.log.Printf("[TRACE] Skip %q -- not a regular file.\n", path)
		return nil
	}

	var (
		err  error
		info fs.FileInfo
	)

	if info, err = d.Info(); err != nil {
		s.log.Printf("[ERROR] Cannot read Info for %s: %s\n",
			path,
			err.Error())
		return err
	} else if info.Size() < minSize {
		s.log.Printf("[TRACE] Skip %q -- too small (%s)\n",
			path,
			krylib.FmtBytes(info.Size()))
		return nil
	}

	s.fileQ <- path
	return nil
} // func (s *Scanner) visitFileFunc(path string, d fs.DirEntry, e error) error

func (s *Scanner) gatherFiles(db *database.Database, newQ chan<- *objects.File) {
	var (
		ticker     *time.Ticker
		knownFiles map[string]bool
	)

	defer db.Close() // nolint: errcheck

	if files, err := s.db.FileGetAll(); err == nil {
		knownFiles = make(map[string]bool, len(files))
		for _, f := range files {
			knownFiles[f.Path] = true
		}
	} else {
		s.log.Printf("[ERROR] Cannot fetch all Files from Database: %s\n",
			err.Error())
		s.Stop()
		return
	}

	ticker = time.NewTicker(timeout)
	defer ticker.Stop()

	for s.Active() {
		select {
		case <-ticker.C:
			continue
		case path := <-s.fileQ:
			var (
				err  error
				file *objects.File
			)

			if knownFiles[path] {
				s.log.Printf("[TRACE] %s is already in database.\n",
					path)
				continue
			} else if file, err = db.FileAdd(path); err != nil {
				s.log.Printf("[ERROR] Cannot add File %s to Database: %s\n",
					path,
					err.Error())
				continue
			}

			knownFiles[path] = true

			if newQ != nil {
				newQ <- file
			}
		}
	}
} // func (s *Scanner) gatherFiles()
