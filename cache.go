// +build !windows !solaris

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
    "git.mills.io/prologic/bitcask"
)

var ErrIterationDone = errors.New("iteration done")

type Cache string

func (c Cache) Put(key, val []byte) error {
    cc, err := bitcask.Open(string(c))
    if err != nil {
        return err
    }
    defer cc.Close()
    return cc.Put(key, val)
}

func (c Cache) Get(key []byte) ([]byte, error) {
    cc, err := bitcask.Open(string(c))
    if err != nil {
        return []byte{}, err
    }
    defer cc.Close()
    
    val, err := cc.Get(key)
    if errors.Is(err, bitcask.ErrKeyNotFound) {
        return nil, nil
    }
    return val, err
}

func (c Cache) Del(key []byte) error {
    cc, err := bitcask.Open(string(c))
    if err != nil {
        return err
    }
    defer cc.Close()
    
    err = cc.Delete(key)
    if errors.Is(err, bitcask.ErrKeyNotFound) {
        return nil
    }
    return err
}

func (c Cache) Fold(fn func(key, val []byte) error) (err error) {
    cc, err := bitcask.Open(string(c))
    if err != nil {
        return err
    }
    defer cc.Close()

    err = cc.Fold(func(key []byte) error {
        val, err := cc.Get(key)
        if err != nil {
            return err
        }

        return fn(key,val)
    })

    if errors.Is(err, ErrIterationDone) {
        return nil
    }
    return err
}
