package main

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

var (
	id        []byte
	fname     = filepath.Join("testdir", "test.txt")
	fname2    = filepath.Join("dirtest", "something-else", fname)
	fnameID   = filepath.Join("dirID", "something-else", fname)
	data      = []byte("test data")
	sha256sum = "916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"
)

func TestSaveData(t *testing.T) {
	id := "randomID"

	f, err := saveData(id, fnameID, data)
	assert.NoError(t, err)
	assert.Equal(t, f.Sha256sum, sha256sum)

	ok, err := exists(filepath.Join(cfg.BaseDir, f.Path))
	assert.NoError(t, err)
	assert.True(t, ok)

	path, err := ccID.Get([]byte(f.ID))
	assert.NoError(t, err)
	assert.Equal(t, path, []byte(fnameID))

	h, err := ccHash.Get([]byte(fnameID))
	assert.NoError(t, err)
	assert.Equal(t, h, []byte(sha256sum))
}

func TestPut(t *testing.T) {
	f, err := put(fname, data)
	assert.NoError(t, err)
	assert.Equal(t, f.Sha256sum, sha256sum)

	id = []byte(f.ID)

	ok, err := exists(filepath.Join(cfg.BaseDir, f.Path))
	assert.NoError(t, err)
	assert.True(t, ok)

	path, err := ccID.Get([]byte(f.ID))
	assert.NoError(t, err)
	assert.Equal(t, path, []byte(fname))

	h, err := ccHash.Get([]byte(fname))
	assert.NoError(t, err)
	assert.Equal(t, h, []byte(sha256sum))
}

func TestMove(t *testing.T) {
	err := move(fname, fname2)
	assert.NoError(t, err)

	absPath := filepath.Join(cfg.BaseDir, fname2)
	ok, err := exists(absPath)
	assert.NoError(t, err)
	assert.True(t, ok)

	path, err := ccID.Get(id)
	assert.NoError(t, err)
	assert.Equal(t, path, []byte(fname2))

	h, err := ccHash.Get([]byte(fname2))
	assert.NoError(t, err)
	assert.Equal(t, sha256sum, string(h))
}

func TestDel(t *testing.T) {
	relPath := filepath.Join("dirtest", "something-else")

	err := del(relPath)
	assert.NoError(t, err)

	ok, err := exists(filepath.Join(cfg.BaseDir, fname2))
	assert.NoError(t, err)
	assert.False(t, ok)

	path, err := ccID.Get(id)
	assert.NoError(t, err)
	assert.Nil(t, path)

	hash, err := ccHash.Get([]byte(fname2))
	assert.NoError(t, err)
	assert.Nil(t, hash)
}

func TestRestore(t *testing.T) {
	var files = []File{
		{"test/file0.txt", "sha256sum", "test_id_0"},
		{"test/file1.txt", "sha256sum", "test_id_1"},
		{"test/file2.txt", "sha256sum", "test_id_2"},
		{"test/file3.txt", "sha256sum", "test_id_3"},
		{"test/file4.txt", "sha256sum", "test_id_4"},
		{"test/file5.txt", "sha256sum", "test_id_5"},
		{"test/file6.txt", "sha256sum", "test_id_6"},
		{"test/file7.txt", "sha256sum", "test_id_7"},
		{"test/file8.txt", "sha256sum", "test_id_8"},
		{"test/file9.txt", "sha256sum", "test_id_9"},
	}

	errs := restore(files)
	assert.True(t, len(errs) == 0, "length of errors not zero")

	for _, f := range files {
		path, err := ccID.Get([]byte(f.ID))
		assert.NoError(t, err)
		assert.NotNil(t, path)

		hash, err := ccHash.Get([]byte(f.Path))
		assert.NoError(t, err)
		assert.NotNil(t, hash)
	}
}

func init() {
	cfg = Config{
		BaseDir:  filepath.Join(Home, ".adam_test"),
		CacheDir: filepath.Join(Home, ".cache", "adam_test"),
		Port:     ":8080",
	}

	createIfNotExists(cfg.BaseDir)
	createIfNotExists(cfg.CacheDir)

	ccHash = Cache(filepath.Join(cfg.CacheDir, "sha256sum"))
	ccID = Cache(filepath.Join(cfg.CacheDir, "ids"))
}
