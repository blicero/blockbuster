// /home/krylon/go/src/github.com/blicero/blockbuster/database/04_person_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-12 18:37:32 krylon>

package database

import (
	"testing"
	"time"

	"github.com/blicero/blockbuster/common"
	"github.com/blicero/blockbuster/objects"
)

var names = []string{
	"Peter Lustig",
	"Dieter Hallervorden",
	"Gert Fröbe",
}

var people []objects.Person

func TestPersonAdd(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var (
		err error
		p   *objects.Person
	)

	for _, n := range names {
		if p, err = tdb.PersonAdd(n, time.Now()); err != nil {
			t.Fatalf("Cannot add Person %s to Database: %s",
				n,
				err.Error())
		} else if p == nil {
			t.Fatalf("PersonAdd(%s) did not return an error, but the Person is nil",
				n)
		} else if p.ID == 0 {
			t.Fatalf("PersonAdd(%s) returned Person without a valid ID",
				n)
		}
	}
} // func TestPersonAdd(t *testing.T)

func TestPersonGetAll(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	var err error

	if people, err = tdb.PersonGetAll(); err != nil {
		t.Fatalf("PersonGetAll failed: %s",
			err.Error())
	} else if len(people) != len(names) {
		defer func() { people = nil }()
		t.Fatalf("PersonGetAll returned unexpected number of values: %d (expected %d)",
			len(people),
			len(names))
	}
} // func TestPersonGetAll(t *testing.T)

func TestPersonGetByID(t *testing.T) {
	if tdb == nil {
		t.SkipNow()
	}

	for _, p := range people {
		var (
			err error
			res *objects.Person
		)

		if res, err = tdb.PersonGetByID(p.ID); err != nil {
			t.Fatalf("Error getting Person %s by ID (%d): %s",
				p.Name,
				p.ID,
				err.Error())
		} else if res == nil {
			t.Fatalf("PersonGetByID did not find %s (%d)",
				p.Name,
				p.ID)
		} else if res.Name != p.Name {
			t.Fatalf("PersonGetByID(%d) returned wrong name: %s (expected %s)",
				p.ID,
				res.Name,
				p.Name)
		} else if !common.TimeEqual(res.Birthday, p.Birthday) {
			t.Fatalf(`PersonGetByID(%d) return wrong Birthday!
Got:      %s
Expected: %s`,
				p.ID,
				res.Birthday.Format(common.TimestampFormat),
				p.Birthday.Format(common.TimestampFormat))
		}
	}
} // func TestPersonGetByID(t *testing.T)
