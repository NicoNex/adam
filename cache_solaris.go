// +build solaris

/*
 * Adam - Adam's Data Access Manager
 * Copyright (C) 2021 Nicolò Santamaria
 *
 * Adam is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Adam is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
)

// ErrIterationDone is useful to stop the iteration in the Fold function.
var ErrIterationDone = errors.New("iteration done")

// Cache is the abstraction object to the key-value database used for caching.
type Cache string

// Put stores a value in the cache.
func (c Cache) Put(key, val []byte) error {
	cc, err := leveldb.OpenFile(string(c), nil)
	if err != nil {
		return err
	}
	defer cc.Close()
	return cc.Put(key, val, nil)
}

// Get returns the value from the cache associated with the given key.
// If no value is associated with the given key nil is returned.
func (c Cache) Get(key []byte) ([]byte, error) {
	cc, err := leveldb.OpenFile(string(c), nil)
	if err != nil {
		return []byte{}, err
	}
	defer cc.Close()
	return cc.Get(key, nil)
}

// Del deletes the value in the cache that corresponds to the given key.
func (c Cache) Del(key []byte) error {
	cc, err := leveldb.OpenFile(string(c), nil)
	if err != nil {
		return err
	}
	defer cc.Close()
	return cc.Delete(key, nil)
}

// Fold iterates over all the key-value pairs stored in the cache and calls the 
// function in input passing those values as argument.
func (c Cache) Fold(fn func(key, val []byte) error) (err error) {
	cc, err := leveldb.OpenFile(string(c), nil)
	if err != nil {
		return err
	}
	defer cc.Close()

	iter := cc.NewIterator(nil, nil)

	for iter.Next() {
		if err := fn(iter.Key(), iter.Value()); err != nil {
			break
		}
	}
	iter.Release()

	return iter.Error()
}
