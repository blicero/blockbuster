// /home/krylon/go/src/github.com/blicero/blockbuster/database/database.go
// -*- mode: go; coding: utf-8; -*-
// Created on 02. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-14 20:20:57 krylon>

// Package database is wrapper around the actual database connection.
// For the time being, we use SQLite, because it is awesome.
package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/blicero/blockbuster/common"
	"github.com/blicero/blockbuster/database/query"
	"github.com/blicero/blockbuster/logdomain"
	"github.com/blicero/blockbuster/objects"
	"github.com/blicero/krylib"
	_ "github.com/mattn/go-sqlite3" // Import the database driver
)

var (
	openLock sync.Mutex
	idCnt    int64
)

// ErrTxInProgress indicates that an attempt to initiate a transaction failed
// because there is already one in progress.
var ErrTxInProgress = errors.New("A Transaction is already in progress")

// ErrNoTxInProgress indicates that an attempt was made to finish a
// transaction when none was active.
var ErrNoTxInProgress = errors.New("There is no transaction in progress")

// ErrEmptyUpdate indicates that an update operation would not change any
// values.
var ErrEmptyUpdate = errors.New("Update operation does not change any values")

// ErrInvalidValue indicates that one or more parameters passed to a method
// had values that are invalid for that operation.
var ErrInvalidValue = errors.New("Invalid value for parameter")

// ErrObjectNotFound indicates that an Object was not found in the database.
var ErrObjectNotFound = errors.New("object was not found in database")

// ErrInvalidSavepoint is returned when a user of the Database uses an unkown
// (or expired) savepoint name.
var ErrInvalidSavepoint = errors.New("that save point does not exist")

// If a query returns an error and the error text is matched by this regex, we
// consider the error as transient and try again after a short delay.
var retryPat = regexp.MustCompile("(?i)database is (?:locked|busy)")

// worthARetry returns true if an error returned from the database
// is matched by the retryPat regex.
func worthARetry(e error) bool {
	return retryPat.MatchString(e.Error())
} // func worthARetry(e error) bool

// retryDelay is the amount of time we wait before we repeat a database
// operation that failed due to a transient error.
const retryDelay = 25 * time.Millisecond

func waitForRetry() {
	time.Sleep(retryDelay)
} // func waitForRetry()

// Database is the storage backend for managing Feeds and news.
//
// It is not safe to share a Database instance between goroutines, however
// opening multiple connections to the same Database is safe.
type Database struct {
	id            int64
	db            *sql.DB
	tx            *sql.Tx
	log           *log.Logger
	path          string
	spNameCounter int
	spNameCache   map[string]string
	queries       map[query.ID]*sql.Stmt
}

// Open opens a Database. If the database specified by the path does not exist,
// yet, it is created and initialized.
func Open(path string) (*Database, error) {
	var (
		err      error
		dbExists bool
		db       = &Database{
			path:          path,
			spNameCounter: 1,
			spNameCache:   make(map[string]string),
			queries:       make(map[query.ID]*sql.Stmt),
		}
	)

	openLock.Lock()
	defer openLock.Unlock()
	idCnt++
	db.id = idCnt

	if db.log, err = common.GetLogger(logdomain.Database); err != nil {
		return nil, err
	} else if common.Debug {
		db.log.Printf("[DEBUG] Open database %s\n", path)
	}

	var connstring = fmt.Sprintf("%s?_locking=NORMAL&_journal=WAL&_fk=1&recursive_triggers=0",
		path)

	if dbExists, err = krylib.Fexists(path); err != nil {
		db.log.Printf("[ERROR] Failed to check if %s already exists: %s\n",
			path,
			err.Error())
		return nil, err
	} else if db.db, err = sql.Open("sqlite3", connstring); err != nil {
		db.log.Printf("[ERROR] Failed to open %s: %s\n",
			path,
			err.Error())
		return nil, err
	}

	if !dbExists {
		if err = db.initialize(); err != nil {
			var e2 error
			if e2 = db.db.Close(); e2 != nil {
				db.log.Printf("[CRITICAL] Failed to close database: %s\n",
					e2.Error())
				return nil, e2
			} else if e2 = os.Remove(path); e2 != nil {
				db.log.Printf("[CRITICAL] Failed to remove database file %s: %s\n",
					db.path,
					e2.Error())
			}
			return nil, err
		}
		db.log.Printf("[INFO] Database at %s has been initialized\n",
			path)
	}

	return db, nil
} // func Open(path string) (*Database, error)

func (db *Database) initialize() error {
	var err error
	var tx *sql.Tx

	if common.Debug {
		db.log.Printf("[DEBUG] Initialize fresh database at %s\n",
			db.path)
	}

	if tx, err = db.db.Begin(); err != nil {
		db.log.Printf("[ERROR] Cannot begin transaction: %s\n",
			err.Error())
		return err
	}

	for _, q := range initQueries {
		db.log.Printf("[TRACE] Execute init query:\n%s\n",
			q)
		if _, err = tx.Exec(q); err != nil {
			db.log.Printf("[ERROR] Cannot execute init query: %s\n%s\n",
				err.Error(),
				q)
			if rbErr := tx.Rollback(); rbErr != nil {
				db.log.Printf("[CANTHAPPEN] Cannot rollback transaction: %s\n",
					rbErr.Error())
				return rbErr
			}
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		db.log.Printf("[CANTHAPPEN] Failed to commit init transaction: %s\n",
			err.Error())
		return err
	}

	return nil
} // func (db *Database) initialize() error

// Close closes the database.
// If there is a pending transaction, it is rolled back.
func (db *Database) Close() error {
	// I wonder if would make more snese to panic() if something goes wrong

	var err error

	if db.tx != nil {
		if err = db.tx.Rollback(); err != nil {
			db.log.Printf("[CRITICAL] Cannot roll back pending transaction: %s\n",
				err.Error())
			return err
		}
		db.tx = nil
	}

	for key, stmt := range db.queries {
		if err = stmt.Close(); err != nil {
			db.log.Printf("[CRITICAL] Cannot close statement handle %s: %s\n",
				key,
				err.Error())
			return err
		}
		delete(db.queries, key)
	}

	if err = db.db.Close(); err != nil {
		db.log.Printf("[CRITICAL] Cannot close database: %s\n",
			err.Error())
	}

	db.db = nil
	return nil
} // func (db *Database) Close() error

func (db *Database) getQuery(id query.ID) (*sql.Stmt, error) {
	var (
		stmt  *sql.Stmt
		found bool
		err   error
	)

	if stmt, found = db.queries[id]; found {
		return stmt, nil
	} else if _, found = dbQueries[id]; !found {
		return nil, fmt.Errorf("Unknown Query %d",
			id)
	}

	db.log.Printf("[TRACE] Prepare query %s\n", id)

PREPARE_QUERY:
	if stmt, err = db.db.Prepare(dbQueries[id]); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto PREPARE_QUERY
		}

		db.log.Printf("[ERROR] Cannor parse query %s: %s\n%s\n",
			id,
			err.Error(),
			dbQueries[id])
		return nil, err
	}

	db.queries[id] = stmt
	return stmt, nil
} // func (db *Database) getQuery(query.ID) (*sql.Stmt, error)

func (db *Database) resetSPNamespace() {
	db.spNameCounter = 1
	db.spNameCache = make(map[string]string)
} // func (db *Database) resetSPNamespace()

func (db *Database) generateSPName(name string) string {
	var spname = fmt.Sprintf("Savepoint%05d",
		db.spNameCounter)

	db.spNameCache[name] = spname
	db.spNameCounter++
	return spname
} // func (db *Database) generateSPName() string

// PerformMaintenance performs some maintenance operations on the database.
// It cannot be called while a transaction is in progress and will block
// pretty much all access to the database while it is running.
func (db *Database) PerformMaintenance() error {
	var mQueries = []string{
		"PRAGMA wal_checkpoint(TRUNCATE)",
		"VACUUM",
		"REINDEX",
		"ANALYZE",
	}
	var err error

	if db.tx != nil {
		return ErrTxInProgress
	}

	for _, q := range mQueries {
		if _, err = db.db.Exec(q); err != nil {
			db.log.Printf("[ERROR] Failed to execute %s: %s\n",
				q,
				err.Error())
		}
	}

	return nil
} // func (db *Database) PerformMaintenance() error

// Begin begins an explicit database transaction.
// Only one transaction can be in progress at once, attempting to start one,
// while another transaction is already in progress will yield ErrTxInProgress.
func (db *Database) Begin() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Begin Transaction\n",
		db.id)

	if db.tx != nil {
		return ErrTxInProgress
	}

BEGIN_TX:
	for db.tx == nil {
		if db.tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				continue BEGIN_TX
			} else {
				db.log.Printf("[ERROR] Failed to start transaction: %s\n",
					err.Error())
				return err
			}
		}
	}

	db.resetSPNamespace()

	return nil
} // func (db *Database) Begin() error

// SavepointCreate creates a savepoint with the given name.
//
// Savepoints only make sense within a running transaction, and just like
// with explicit transactions, managing them is the responsibility of the
// user of the Database.
//
// Creating a savepoint without a surrounding transaction is not allowed,
// even though SQLite allows it.
//
// For details on how Savepoints work, check the excellent SQLite
// documentation, but here's a quick guide:
//
// Savepoints are kind-of-like transactions within a transaction: One
// can create a savepoint, make some changes to the database, and roll
// back to that savepoint, discarding all changes made between
// creating the savepoint and rolling back to it. Savepoints can be
// quite useful, but there are a few things to keep in mind:
//
// - Savepoints exist within a transaction. When the surrounding transaction
//   is finished, all savepoints created within that transaction cease to exist,
//   no matter if the transaction is commited or rolled back.
//
// - When the database is recovered after being interrupted during a
//   transaction, e.g. by a power outage, the entire transaction is rolled back,
//   including all savepoints that might exist.
//
// - When a savepoint is released, nothing changes in the state of the
//   surrounding transaction. That means rolling back the surrounding
//   transaction rolls back the entire transaction, regardless of any
//   savepoints within.
//
// - Savepoints do not nest. Releasing a savepoint releases it and *all*
//   existing savepoints that have been created before it. Rolling back to a
//   savepoint removes that savepoint and all savepoints created after it.
func (db *Database) SavepointCreate(name string) error {
	var err error

	db.log.Printf("[DEBUG] SavepointCreate(%s)\n",
		name)

	if db.tx == nil {
		return ErrNoTxInProgress
	}

SAVEPOINT:
	// It appears that the SAVEPOINT statement does not support placeholders.
	// But I do want to used named savepoints.
	// And I do want to use the given name so that no SQL injection
	// becomes possible.
	// It would be nice if the database package or at least the SQLite
	// driver offered a way to escape the string properly.
	// One possible solution would be to use names generated by the
	// Database instead of user-defined names.
	//
	// But then I need a way to use the Database-generated name
	// in rolling back and releasing the savepoint.
	// I *could* use the names strictly inside the Database, store them in
	// a map or something and hand out a key to that name to the user.
	// Since savepoint only exist within one transaction, I could even
	// re-use names from one transaction to the next.
	//
	// Ha! I could accept arbitrary names from the user, generate a
	// clean name, and store these in a map. That way the user can
	// still choose names that are outwardly visible, but they do
	// not touch the Database itself.
	//
	//if _, err = db.tx.Exec("SAVEPOINT ?", name); err != nil {
	// if _, err = db.tx.Exec("SAVEPOINT " + name); err != nil {
	// 	if worthARetry(err) {
	// 		waitForRetry()
	// 		goto SAVEPOINT
	// 	}

	// 	db.log.Printf("[ERROR] Failed to create savepoint %s: %s\n",
	// 		name,
	// 		err.Error())
	// }

	var internalName = db.generateSPName(name)

	var spQuery = "SAVEPOINT " + internalName

	if _, err = db.tx.Exec(spQuery); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto SAVEPOINT
		}

		db.log.Printf("[ERROR] Failed to create savepoint %s: %s\n",
			name,
			err.Error())
	}

	return err
} // func (db *Database) SavepointCreate(name string) error

// SavepointRelease releases the Savepoint with the given name, and all
// Savepoints created before the one being release.
func (db *Database) SavepointRelease(name string) error {
	var (
		err                   error
		internalName, spQuery string
		validName             bool
	)

	db.log.Printf("[DEBUG] SavepointRelease(%s)\n",
		name)

	if db.tx != nil {
		return ErrNoTxInProgress
	}

	if internalName, validName = db.spNameCache[name]; !validName {
		db.log.Printf("[ERROR] Attempt to release unknown Savepoint %q\n",
			name)
		return ErrInvalidSavepoint
	}

	db.log.Printf("[DEBUG] Release Savepoint %q (%q)",
		name,
		db.spNameCache[name])

	spQuery = "RELEASE SAVEPOINT " + internalName

SAVEPOINT:
	if _, err = db.tx.Exec(spQuery); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto SAVEPOINT
		}

		db.log.Printf("[ERROR] Failed to release savepoint %s: %s\n",
			name,
			err.Error())
	} else {
		delete(db.spNameCache, internalName)
	}

	return err
} // func (db *Database) SavepointRelease(name string) error

// SavepointRollback rolls back the running transaction to the given savepoint.
func (db *Database) SavepointRollback(name string) error {
	var (
		err                   error
		internalName, spQuery string
		validName             bool
	)

	db.log.Printf("[DEBUG] SavepointRollback(%s)\n",
		name)

	if db.tx != nil {
		return ErrNoTxInProgress
	}

	if internalName, validName = db.spNameCache[name]; !validName {
		return ErrInvalidSavepoint
	}

	spQuery = "ROLLBACK TO SAVEPOINT " + internalName

SAVEPOINT:
	if _, err = db.tx.Exec(spQuery); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto SAVEPOINT
		}

		db.log.Printf("[ERROR] Failed to create savepoint %s: %s\n",
			name,
			err.Error())
	}

	delete(db.spNameCache, name)
	return err
} // func (db *Database) SavepointRollback(name string) error

// Rollback terminates a pending transaction, undoing any changes to the
// database made during that transaction.
// If no transaction is active, it returns ErrNoTxInProgress
func (db *Database) Rollback() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Roll back Transaction\n",
		db.id)

	if db.tx == nil {
		return ErrNoTxInProgress
	} else if err = db.tx.Rollback(); err != nil {
		return fmt.Errorf("Cannot roll back database transaction: %s",
			err.Error())
	}

	db.tx = nil
	db.resetSPNamespace()

	return nil
} // func (db *Database) Rollback() error

// Commit ends the active transaction, making any changes made during that
// transaction permanent and visible to other connections.
// If no transaction is active, it returns ErrNoTxInProgress
func (db *Database) Commit() error {
	var err error

	db.log.Printf("[DEBUG] Database#%d Commit Transaction\n",
		db.id)

	if db.tx == nil {
		return ErrNoTxInProgress
	} else if err = db.tx.Commit(); err != nil {
		return fmt.Errorf("Cannot commit transaction: %s",
			err.Error())
	}

	db.resetSPNamespace()
	db.tx = nil
	return nil
} // func (db *Database) Commit() error

// FolderAdd adds a Folder to the Database.
func (db *Database) FolderAdd(path string) (*objects.Folder, error) {
	const qid query.ID = query.FolderAdd
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return nil, err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return nil, errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var res sql.Result

EXEC_QUERY:
	if res, err = stmt.Exec(path); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add File %s to database: %s",
				path,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return nil, err
		}
	} else {
		var folderID int64

		if folderID, err = res.LastInsertId(); err != nil {
			db.log.Printf("[ERROR] Cannot get ID of new Feed %s: %s\n",
				path,
				err.Error())
			return nil, err
		}

		status = true
		return &objects.Folder{
			ID:       folderID,
			Path:     path,
			LastScan: time.Unix(0, 0),
		}, nil
	}
} // func (db *Database) FolderAdd(path string) (*object.Folder, error)

// FolderRemove removes a Folder from the Database.
// All Files from that Folder have to be removed from the Database before the
// Folder itself can be removed.
func (db *Database) FolderRemove(f *objects.Folder) error {
	const qid query.ID = query.FolderRemove
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(f.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot delete Folder %s (%d) from database: %s",
				f.Path,
				f.ID,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) FolderRemove(f *objects.Folder) error

// FolderUpdateScan updates the timestamp for the given Folder's last scan.
func (db *Database) FolderUpdateScan(f *objects.Folder, stamp time.Time) error {
	const qid query.ID = query.FolderUpdateScan
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(stamp.Unix(), f.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot delete Folder %s (%d) from database: %s",
				f.Path,
				f.ID,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	f.LastScan = stamp
	return nil
} // func (db *Database) FolderUpdateScan(f *objects.Folder, stamp time.Time) error

// FolderGetAll fetches all Folders from the Database.
func (db *Database) FolderGetAll() ([]objects.Folder, error) {
	const qid query.ID = query.FolderGetAll
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var list = make([]objects.Folder, 0, 256)

	for rows.Next() {
		var (
			f     objects.Folder
			stamp int64
		)

		if err = rows.Scan(&f.ID, &f.Path, &stamp); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		}

		f.LastScan = time.Unix(stamp, 0)
		list = append(list, f)
	}

	return list, nil
} // func (db *Database) FolderGetAll() ([]objects.Folder, error)

// FolderGetByPath looks up a Folder by its Path
func (db *Database) FolderGetByPath(path string) (*objects.Folder, error) {
	const qid query.ID = query.FolderGetByPath
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(path); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			f     = &objects.Folder{Path: path}
			stamp int64
		)

		if err = rows.Scan(&f.ID, &stamp); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		}

		f.LastScan = time.Unix(stamp, 0)
		return f, nil
	}

	return nil, nil
} // func (db *Database) FolderGetByPath(path string) (*objects.Folder, error)

// FileAdd registers a File with the Database.
func (db *Database) FileAdd(path string, folder *objects.Folder) (*objects.File, error) {
	const qid query.ID = query.FileAdd
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return nil, err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return nil, errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var res sql.Result

EXEC_QUERY:
	if res, err = stmt.Exec(path, folder.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add File %s to database: %s",
				path,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return nil, err
		}
	} else {
		var fileID int64

		if fileID, err = res.LastInsertId(); err != nil {
			db.log.Printf("[ERROR] Cannot get ID of new Feed %s: %s\n",
				path,
				err.Error())
			return nil, err
		}

		status = true
		return &objects.File{
			ID:       fileID,
			FolderID: folder.ID,
			Path:     path,
		}, nil
	}
} // func (db *Database) FileAdd(path string) (*objects.File, error)

// FileRemove deletes a File from the Database.
func (db *Database) FileRemove(f *objects.File) error {
	const qid query.ID = query.FileRemove
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(f.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot delete File %s (%d) from database: %s",
				f.Path,
				f.ID,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) FileRemove(f *objects.File) error

// FileGetAll retrieves all registered Files from the Database.
func (db *Database) FileGetAll() ([]objects.File, error) {
	const qid query.ID = query.FileGetAll

	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var list = make([]objects.File, 0, 256)

	for rows.Next() {
		var (
			f     objects.File
			title *string
			year  *int64
		)

		if err = rows.Scan(&f.ID, &f.FolderID, &f.Path, &title, &year, &f.Hidden); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		}

		if title != nil {
			f.Title = *title
		}

		if year != nil {
			f.Year = *year
		}

		list = append(list, f)
	}

	return list, nil
} // func (db *Database) FileGetAll() ([]objects.File, error)

// FileGetByPath retrieves a File objects by its path in the file system.
func (db *Database) FileGetByPath(path string) (*objects.File, error) {
	const qid query.ID = query.FileGetByPath
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(path); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			f = &objects.File{Path: path}
		)

		if err = rows.Scan(&f.ID, &f.FolderID, &f.Title, &f.Year, &f.Hidden); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		}

		return f, nil
	}

	return nil, nil
} // func (db *Database) FileGetByPath(path string) (*objects.File, error)

// FileGetByID loads a File by its ID.
func (db *Database) FileGetByID(id int64) (*objects.File, error) {
	const qid query.ID = query.FileGetByID
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			f = &objects.File{ID: id}
		)

		if err = rows.Scan(&f.FolderID, &f.Path, &f.Title, &f.Year, &f.Hidden); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		}

		return f, nil
	}

	return nil, nil
} // func (db *Database) FileGetByID(id int64) (*objects.File, error)

// TagAdd adds a new Tag to the Database.
func (db *Database) TagAdd(name string) (*objects.Tag, error) {
	const qid query.ID = query.TagAdd
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return nil, err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return nil, errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var res sql.Result

EXEC_QUERY:
	if res, err = stmt.Exec(name); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Tag %s to database: %s",
				name,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return nil, err
		}
	} else {
		var tagID int64

		if tagID, err = res.LastInsertId(); err != nil {
			db.log.Printf("[ERROR] Cannot get ID of new Feed %s: %s\n",
				name,
				err.Error())
			return nil, err
		}

		status = true
		return &objects.Tag{
			ID:   tagID,
			Name: name,
		}, nil
	}
} // func (db *Database) TagAdd(name string) (*objects.Tag, error)

// TagDelete removes a Tag from the Database.
// NB, all the links pointing to the Tag have to be deleted before the Tag itself can be
// deleted.
func (db *Database) TagDelete(t *objects.Tag) error {
	const qid query.ID = query.TagDelete
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(t.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot delete Tag %s from database: %s",
				t.Name,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) TagDelete(t *objects.Tag) error

// TagGetAll fetches all Tags from the Database.
func (db *Database) TagGetAll() ([]objects.Tag, error) {
	const qid query.ID = query.TagGetAll
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var list = make([]objects.Tag, 0, 32)

	for rows.Next() {
		var (
			t objects.Tag
		)

		if err = rows.Scan(&t.ID, &t.Name); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		}

		list = append(list, t)
	}

	return list, nil
} // func (db *Database) TagGetAll() ([]objects.Tag, error)

// TagLinkAdd links the given Tag to the given File
func (db *Database) TagLinkAdd(f *objects.File, t *objects.Tag) error {
	const qid query.ID = query.TagLinkAdd
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(f.ID, t.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Tag %s to File %s: %s",
				t.Name,
				f.DisplayTitle(),
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) TagLinkAdd(f *objects.File, t *objects.Tag) error

// TagLinkDelete removes the link between the given Tag and File.
func (db *Database) TagLinkDelete(f *objects.File, t *objects.Tag) error {
	const qid query.ID = query.TagLinkDelete
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(f.ID, t.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot remove link of Tag %s to File %s: %s",
				t.Name,
				f.DisplayTitle(),
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) TagLinkDelete(f *objects.File, t *objects.Tag) error

// TagLinkGetByTag fetches all Files linked to the given Tag.
func (db *Database) TagLinkGetByTag(t *objects.Tag) ([]objects.File, error) {
	const qid query.ID = query.TagLinkGetByTag
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(t.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var files = make([]objects.File, 0, 10)

	for rows.Next() {
		var (
			f objects.File
		)

		if err = rows.Scan(&f.ID, &f.FolderID, &f.Path, &f.Title, &f.Year); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		}

		files = append(files, f)
	}

	return files, nil
} // func (db *Database) TagLinkGetByTag(t *objects.Tag) ([]objects.File, error)

// TagLinkGetByFile loads all Tags linked to the given File.
func (db *Database) TagLinkGetByFile(f *objects.File) (map[int64]objects.Tag, error) {
	const qid query.ID = query.TagLinkGetByFile
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(f.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var tags = make(map[int64]objects.Tag)

	for rows.Next() {
		var (
			t objects.Tag
		)

		if err = rows.Scan(&t.ID, &t.Name); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		}

		tags[t.ID] = t
	}

	return tags, nil
} // func (db *Database) TagLinkGetByFile(f *objects.File) (map[int64]objects.Tag, error)

// PersonAdd adds a new Person to the database.
func (db *Database) PersonAdd(name string, birthday time.Time) (*objects.Person, error) {
	const qid query.ID = query.PersonAdd
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return nil, err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return nil, errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var (
		res    sql.Result
		bstamp int64
	)

	if birthday.IsZero() {
		bstamp = 0
	} else {
		bstamp = birthday.Unix()
	}

EXEC_QUERY:
	if res, err = stmt.Exec(name, bstamp); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Tag %s to database: %s",
				name,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return nil, err
		}
	} else {
		var id int64

		if id, err = res.LastInsertId(); err != nil {
			db.log.Printf("[ERROR] Cannot get ID of new Feed %s: %s\n",
				name,
				err.Error())
			return nil, err
		}

		status = true
		return &objects.Person{
			ID:       id,
			Name:     name,
			Birthday: birthday,
		}, nil
	}
} // func (db *Database) PersonAdd(name string, birthday time.Time) (*objects.Person, error)

// PersonGetAll loads all Persons from the Database, in no particular order.
func (db *Database) PersonGetAll() ([]objects.Person, error) {
	const qid query.ID = query.PersonGetAll
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var list = make([]objects.Person, 0, 32)

	for rows.Next() {
		var (
			p      objects.Person
			bstamp int64
		)

		if err = rows.Scan(&p.ID, &p.Name, &bstamp); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		}

		if bstamp != 0 {
			p.Birthday = time.Unix(bstamp, 0)
		}

		list = append(list, p)
	}

	return list, nil
} // func (db *Database) PersonGetAll() ([]objects.Person, error)

// PersonGetByID looks up a Person by their ID.
func (db *Database) PersonGetByID(id int64) (*objects.Person, error) {
	const qid query.ID = query.PersonGetByID
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	if rows.Next() {
		var (
			p      = &objects.Person{ID: id}
			bstamp int64
		)

		if err = rows.Scan(&p.Name, &bstamp); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		}

		if bstamp != 0 {
			p.Birthday = time.Unix(bstamp, 0)
		}

		return p, nil
	}

	return nil, nil
} // func (db *Database) PersonGetByID(id int64) (*objects.Person, error)

// PersonURLAdd attaches a Link to a Person.
func (db *Database) PersonURLAdd(p *objects.Person, l *objects.Link) error {
	const qid query.ID = query.PersonURLAdd
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)
	var (
		res sql.Result
		id  int64
	)

EXEC_QUERY:
	if res, err = stmt.Exec(p.ID, l.URL.String(), l.Title, l.Description); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Link %s to Person %s: %s",
				l.DisplayTitle(),
				p.Name,
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	} else if id, err = res.LastInsertId(); err != nil {
		db.log.Printf("[ERROR] Cannot get ID of newly added Link %s: %s\n",
			l.DisplayTitle(),
			err.Error())
		return err
	}

	l.ID = id
	status = true
	return nil
} // func (db *Database) PersonURLAdd(p *objects.Person, l *objects.Link) error

// PersonURLDelete deletes a Link that has been attached to a Person
func (db *Database) PersonURLDelete(l *objects.Link) error {
	const qid query.ID = query.PersonURLDelete
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(l.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot delete Link %s: %s",
				l.DisplayTitle(),
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) PersonURLDelete(l *objects.Link) error

// PersonURLGetByPerson returns all Links attached to the given Person.
func (db *Database) PersonURLGetByPerson(p *objects.Person) ([]objects.Link, error) {
	const qid query.ID = query.PersonURLGetByPerson
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(p.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var links = make([]objects.Link, 0, 4)

	for rows.Next() {
		var (
			l    objects.Link
			ustr string
		)

		if err = rows.Scan(&l.ID, &ustr, &l.Title, &l.Description); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		} else if l.URL, err = url.Parse(ustr); err != nil {
			db.log.Printf("[ERROR] Cannot parse URL %q: %s\n",
				ustr,
				err.Error())
			return nil, err
		}

		links = append(links, l)
	}

	return links, nil
} // func (db *Database) PersonURLGetByPerson(p *objects.Person) ([]objects.Link, error)

// ActorAdd adds a Person to a File as an actor/actress.
func (db *Database) ActorAdd(f *objects.File, p *objects.Person) error {
	const qid query.ID = query.ActorAdd
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(f.ID, p.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot add Actor %s to Film %s: %s",
				p.Name,
				f.DisplayTitle(),
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) ActorAdd(f *objects.File, p *objects.Person) error

// ActorDelete removes a Person from a Files "acting credits".
func (db *Database) ActorDelete(f *objects.File, p *objects.Person) error {
	const qid query.ID = query.ActorDelete
	var (
		err    error
		msg    string
		stmt   *sql.Stmt
		tx     *sql.Tx
		status bool
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid.String(),
			err.Error())
		return err
	} else if db.tx != nil {
		tx = db.tx
	} else {
	BEGIN_AD_HOC:
		if tx, err = db.db.Begin(); err != nil {
			if worthARetry(err) {
				waitForRetry()
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting transaction: %s\n",
					err.Error())
				db.log.Printf("[ERROR] %s\n", msg)
				return errors.New(msg)
			}

		} else {
			defer func() {
				var err2 error
				if status {
					if err2 = tx.Commit(); err2 != nil {
						db.log.Printf("[ERROR] Failed to commit ad-hoc transaction: %s\n",
							err2.Error())
					}
				} else if err2 = tx.Rollback(); err2 != nil {
					db.log.Printf("[ERROR] Rollback of ad-hoc transaction failed: %s\n",
						err2.Error())
				}
			}()
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(f.ID, p.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		} else {
			err = fmt.Errorf("Cannot remove Actor %s from Film %s: %s",
				p.Name,
				f.DisplayTitle(),
				err.Error())
			db.log.Printf("[ERROR] %s\n", err.Error())
			return err
		}
	}

	status = true
	return nil
} // func (db *Database) ActorDelete(f *objects.File, p *objects.Person) error

// ActorGetByPerson gets all the Files the given Person has acted in.
func (db *Database) ActorGetByPerson(p *objects.Person) ([]objects.File, error) {
	const qid query.ID = query.ActorGetByPerson
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(p.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var files = make([]objects.File, 0, 10)

	for rows.Next() {
		var (
			f objects.File
		)

		if err = rows.Scan(&f.ID, &f.FolderID, &f.Path, &f.Title, &f.Year); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		}

		files = append(files, f)
	}

	return files, nil
} // func (db *Database) ActorGetByPerson(p *objects.Person) ([]objects.File, error)

// ActorGetByFile gets all the People that have acted in the given File.
func (db *Database) ActorGetByFile(f *objects.File) ([]objects.Person, error) {
	const qid query.ID = query.ActorGetByFile
	var (
		err  error
		stmt *sql.Stmt
	)

	if stmt, err = db.getQuery(qid); err != nil {
		db.log.Printf("[ERROR] Cannot prepare query %s: %s\n",
			qid,
			err.Error())
		return nil, err
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(f.ID); err != nil {
		if worthARetry(err) {
			waitForRetry()
			goto EXEC_QUERY
		}

		return nil, err
	}

	defer rows.Close() // nolint: errcheck,gosec

	var people = make([]objects.Person, 0, 10)

	for rows.Next() {
		var (
			p      objects.Person
			bstamp int64
		)

		if err = rows.Scan(&p.ID, &p.Name, &bstamp); err != nil {
			db.log.Printf("[ERROR] Cannot scan row: %s\n", err.Error())
			return nil, err
		}

		if bstamp != 0 {
			p.Birthday = time.Unix(bstamp, 0)
		}

		people = append(people, p)
	}

	return people, nil
} // func (db *Database) ActorGetByFile(f *objects.File) ([]objects.Person, error)
