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
