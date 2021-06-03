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
