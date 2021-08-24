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

var ErrIterationDone = errors.New("iteration done")

type Cache string

func (c Cache) Put(key, val []byte) error {
	cc, err := leveldb.OpenFile(string(c), nil)
	if err != nil {
		return err
	}
	defer cc.Close()
	return cc.Put(key, val, nil)
}

func (c Cache) Get(key []byte) ([]byte, error) {
	cc, err := leveldb.OpenFile(string(c), nil)
	if err != nil {
		return []byte{}, err
	}
	defer cc.Close()
	return cc.Get(key, nil)
}

func (c Cache) Del(key []byte) error {
	cc, err := leveldb.OpenFile(string(c), nil)
	if err != nil {
		return err
	}
	defer cc.Close()
	return cc.Delete(key, nil)
}

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
