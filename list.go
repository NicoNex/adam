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

import "sync"

type StrList struct {
	s []string
	sync.Mutex
}

func (s *StrList) Append(a ...string) {
	s.Lock()
	s.s = append(s.s, a...)
	s.Unlock()
}

func (s *StrList) Slice() []string {
	return s.s
}

type ErrList struct {
	e []error
	sync.Mutex
}

func (e *ErrList) Append(a ...error) {
	e.Lock()
	e.e = append(e.e, a...)
	e.Unlock()
}

func (e *ErrList) Slice() []error {
	return e.e
}

type FileList struct {
	f []File
	sync.Mutex
}

func (f *FileList) Append(a ...File) {
	f.Lock()
	f.f = append(f.f, a...)
	f.Unlock()
}

func (f *FileList) Slice() []File {
	return f.f
}
