package main

import "sync"

type FileList struct {
	s []File
	sync.Mutex
}

func (f *FileList) Append(a ...File) {
	f.Lock()
	f.s = append(f.s, a...)
	f.Unlock()
}

func (f *FileList) Slice() []File {
	return f.s
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
