package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"io"
	"path/filepath"
	"testing"
)

var (
	testdir  = "testdir"
	testfile = filepath.Join(testdir, "testFile.txt")
	movefile = filepath.Join(testdir, "testFile2.txt")
	testCC   = Cache(filepath.Join(testdir, "cache"))
)

func TestSafeFile(t *testing.T) {
	err := saveFile(testfile, []byte("test data"))
	assert.NoError(t, err)

	ok, err := exists(testfile)
	assert.True(t, ok)
	assert.NoError(t, err)
}

func TestExists(t *testing.T) {
	ok, err := exists(testfile)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestSaveSha256sum(t *testing.T) {
	ccHash = testCC
	h := sha256.New()
	io.Copy(h, bytes.NewReader([]byte("test data")))
	h1 := hex.EncodeToString(h.Sum(nil))

	h2, err := saveSha256sum(testfile)
	assert.NoError(t, err)
	assert.Equal(t, h1, h2)
}

func TestGetSha256Sum(t *testing.T) {
	h := sha256.New()
	io.Copy(h, bytes.NewReader([]byte("test data")))
	h1 := hex.EncodeToString(h.Sum(nil))

	h2, err := getSha256sum(testfile)
	assert.NoError(t, err)
	assert.Equal(t, h1, h2)
}

func TestMoveSha256sum(t *testing.T) {
	h1, _ := getSha256sum(testfile)
	assert.NotEqual(t, h1, "")

	err := moveSha256sum(testfile, movefile)
	assert.NoError(t, err)

	h2, _ := getSha256sum(movefile)
	assert.Equal(t, h1, h2)
}
