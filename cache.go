/*
 * Adam
 * Copyright (C) 2021 Nicolò Santamaria
 *
 * Adam is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
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

import "github.com/prologic/bitcask"

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
	return cc.Get(key)
}

func (c Cache) Del(key []byte) error {
	cc, err := bitcask.Open(string(c))
	if err != nil {
		return err
	}
	defer cc.Close()
	return cc.Delete(key)
}
