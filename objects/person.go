// /home/krylon/go/src/github.com/blicero/blockbuster/objects/person.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-24 22:27:26 krylon>

package objects

import (
	"time"

	"github.com/blicero/blockbuster/common"
)

// Person is ... a Person that can be linked to File as a Director
// or Actor/Actress.
type Person struct {
	ID       int64
	Name     string
	Birthday time.Time
}

// BDayString returns the Person's birthday as an ISO 8601-formatted string.
func (p *Person) BDayString() string {
	return p.Birthday.Format(common.TimestampFormatDate)
} // func (p *Person) BDayString() string
