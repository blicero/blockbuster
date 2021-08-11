// /home/krylon/go/src/github.com/blicero/blockbuster/objects/person.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 08. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2021-08-10 18:26:42 krylon>

package objects

import "time"

// Person is ... a Person that can be linked to File as a Director
// or Actor/Actress.
type Person struct {
	ID       int64
	Name     string
	Birthday time.Time
}
