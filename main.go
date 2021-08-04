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

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
)

var (
	cfg    Config
	ccHash Cache
	ccID   Cache
)

// Returns the Base object with ok=false and the error message encoded in Json.
func errorf(format string, a ...interface{}) string {
	base := Base{false, fmt.Sprintf(format, a...)}
	b, err := json.Marshal(base)
	if err != nil {
		log.Println("errorf", "json.Marshal", err)
		return "internal server error, check logs for details"
	}
	return string(b)
}

func exists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func saveFile(fpath string, content []byte) error {
	var dir = filepath.Dir(fpath)

	// If the directory doesn't exist we create it.
	if ok, err := exists(dir); !ok && err == nil {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := os.WriteFile(fpath, content, 0644); err != nil {
		return err
	}
	return nil
}

func saveSha256sum(fpath string, cnt []byte) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, bytes.NewReader(cnt)); err != nil {
		return "", err
	}

	encHex := hex.EncodeToString(h.Sum(nil))
	if err := ccHash.Put([]byte(fpath), []byte(encHex)); err != nil {
		return "", err
	}
	return encHex, nil
}

func getSha256sum(fpath string) (string, error) {
	c, err := ccHash.Get([]byte(fpath))
	if err != nil {
		return "", err
	}
	return string(c), nil
}

func moveSha256sum(src, dest string) error {
	var s = []byte(src)
	var d = []byte(dest)

	hash, err := ccHash.Get(s)
	if err != nil {
		return err
	}
	if err := ccHash.Del(s); err != nil {
		return err
	}
	if err := ccHash.Put(d, hash); err != nil {
		return err
	}
	return nil
}

func findIDFromPath(path string) (string, error) {
	var id []byte

	return string(id), ccID.Fold(func(key, val []byte) error {
		if string(val) == path {
			id = key
			return ErrIterationDone
		}
		return nil
	})
}

func put(fname string, content []byte) (File, error) {
	var (
		id, hash string
		path     = filepath.Join(cfg.BaseDir, fname)
	)

	// Generate UUID if fname doesn't exist.
	if ok, err := exists(path); err != nil {
		return File{}, fmt.Errorf("put exists: %w", err)
	} else if ok {
		if id, err = findIDFromPath(fname); err != nil {
			return File{}, fmt.Errorf("put findIDFromPath: %w", err)
		}
	} else {
		ident, err := uuid.NewRandom()
		if err != nil {
			return File{}, fmt.Errorf("put uuid.NewRandom: %w", err)
		}
		id = ident.String()
	}

	// Save file to disk.
	if err := saveFile(path, content); err != nil {
		return File{}, fmt.Errorf("put saveFile: %w", err)
	}

	// Save ID to cache.
	if err := ccID.Put([]byte(id), []byte(fname)); err != nil {
		return File{}, fmt.Errorf("put ccID.Put: %w", err)
	}

	// Save sha256sum to cache.
	hash, err := saveSha256sum(fname, content)
	if err != nil {
		return File{}, fmt.Errorf("put saveSha256sum: %w", err)
	}

	return File{ID: id, Sha256sum: hash, Path: fname}, nil
}

func del(fpath string) error {
	var abs = filepath.Join(cfg.BaseDir, fpath)

	if err := os.RemoveAll(abs); err != nil {
		return fmt.Errorf("del os.RemoveAll: %w", err)
	}

	// We delete all the occurrences that have 'fpath' as prefix.
	deletable := make(map[string]string)
	ccID.Fold(func(id, path []byte) error {
		if strings.HasPrefix(string(path), fpath) {
			deletable[string(id)] = string(path)
		}
		return nil
	})

	for id, path := range deletable {
		i := []byte(id)
		p := []byte(path)

		if err := ccHash.Del(p); err != nil {
			log.Println("del", "ccHash.Del", err)
		}
		if err := ccID.Del(i); err != nil {
			log.Println("del", "ccID.Del", err)
		}
	}

	return nil
}

func move(oldpath, newpath string) error {
	var wg sync.WaitGroup

	absDest := filepath.Join(cfg.BaseDir, newpath)
	destDir := filepath.Dir(absDest)

	// Create destination directory if doesn't exist.
	if ok, err := exists(destDir); !ok && err == nil {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("move os.MkdirAll: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("move exists: %w", err)
	}

	absSrc := filepath.Join(cfg.BaseDir, oldpath)

	if err := os.Rename(absSrc, absDest); err != nil {
		return fmt.Errorf("move os.Rename: %w", err)
	}

	// Update the IDs in the cache.
	wg.Add(1)
	go func() {
		defer wg.Done()

		affected := make(map[string]string)
		ccID.Fold(func(k, v []byte) error {
			id := string(k)
			path := string(v)

			if strings.HasPrefix(path, oldpath) {
				affected[id] = strings.Replace(path, oldpath, newpath, -1)
			}

			return nil
		})

		for id, path := range affected {
			if err := ccID.Put([]byte(id), []byte(path)); err != nil {
				log.Println("move", "ccID.Put", err)
			}
		}
	}()

	// Update the checksums of the files.
	wg.Add(1)
	go func() {
		defer wg.Done()

		affected := make(map[string]File)
		ccHash.Fold(func(path, hash []byte) error {
			p := string(path)
			h := string(hash)

			if strings.HasPrefix(p, oldpath) {
				new := strings.Replace(p, oldpath, newpath, -1)
				affected[p] = File{Sha256sum: h, Path: new}
			}
			return nil
		})

		for old, file := range affected {
			if err := ccHash.Del([]byte(old)); err != nil {
				log.Println("move", "ccHash.Del", err)
			}
			if err := ccHash.Put([]byte(file.Path), []byte(file.Sha256sum)); err != nil {
				log.Println("move", "ccHash.Put", err)
			}
		}
	}()

	wg.Wait()
	return nil
}

func restore(files []File) (errs []error) {
	for _, f := range files {
		if err := ccID.Put([]byte(f.ID), []byte(f.Path)); err != nil {
			e := fmt.Errorf("unable to restore ID for %s: %w\n", f.Path, err)
			errs = append(errs, e)
		}
		if err := ccHash.Put([]byte(f.Path), []byte(f.Sha256sum)); err != nil {
			e := fmt.Errorf("unable to restore sha256sum for %s: %w\n", f.Path, err)
			errs = append(errs, e)
		}
	}
	return
}

func restoreFile(fpath string) []error {
	var files []File

	b, err := os.ReadFile(fpath)
	if err != nil {
		return []error{err}
	}

	if err := json.Unmarshal(b, &files); err != nil {
		return []error{err}
	}

	return restore(files)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fmt.Fprintln(w, errorf("invalid request, expected GET got %s", r.Method))
		return
	}

	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		log.Println("handleGet", "url.ParseQuery", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	id := values.Get("id")
	if id == "" {
		fmt.Fprintln(w, errorf("missing id query parameter"))
		return
	}

	path, err := ccID.Get([]byte(id))
	if err != nil {
		log.Println("handleGet", "ccID.Get", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	http.ServeFile(w, r, filepath.Join(cfg.BaseDir, string(path)))
}

func handlePut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fmt.Fprintln(w, errorf("invalid request, expected POST got %s", r.Method))
		return
	}

	// 1Mb in memory the rest on disk.
	r.ParseMultipartForm(1_048_576)
	if r.MultipartForm == nil {
		fmt.Fprintln(w, errorf("no file provided"))
		return
	}

	if len(r.MultipartForm.File) == 0 {
		fmt.Fprintln(w, errorf("no file provided"))
		return
	}

	var (
		wg    sync.WaitGroup
		files FileList
		errs  ErrList
		ok    = true
	)

	for _, headers := range r.MultipartForm.File {
		for _, h := range headers {
			tmp, err := h.Open()
			if err != nil {
				log.Println("handlePut", "h.Open", err)
				continue
			}

			fname := filepath.Base(h.Filename)
			cnt, err := io.ReadAll(tmp)
			tmp.Close()
			if err != nil {
				log.Println("handlePut", "io.ReadAll", err)
				continue
			}

			fdir := strings.TrimPrefix(r.URL.Path, "/put")
			fdir = strings.TrimPrefix(fdir, "/")
			fpath := filepath.Join(fdir, fname)
			wg.Add(1)
			go func(fpath string, cnt []byte) {
				defer wg.Done()

				if file, err := put(fpath, cnt); err == nil {
					files.Append(file)
				} else {
					ok = false
					log.Println("handlePut", err)
					errs.Append(errors.Unwrap(err))
				}
			}(fpath, cnt)
		}
		wg.Wait()

		b, err := json.Marshal(PutResponse{
			Base:   Base{OK: ok},
			Files:  files.Slice(),
			Errors: errs.Slice(),
		})
		if err != nil {
			log.Println("handlePut", "json.Marshal", err)
			fmt.Fprintln(w, errorf(err.Error()))
			return
		}
		fmt.Fprintln(w, string(b))
	}
}

func handleDel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fmt.Fprintln(w, errorf("invalid request, expected GET got %s", r.Method))
		return
	}

	relative := strings.TrimPrefix(r.URL.Path, "/del")

	if relative == "" {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			log.Println("handleDel", "url.ParseQuery", err)
			fmt.Fprintln(w, errorf(err.Error()))
			return
		}

		id := values.Get("id")
		if id == "" {
			fmt.Fprintln(w, errorf("missing id query parameter or path"))
			return
		}

		r, err := ccID.Get([]byte(id))
		if err != nil {
			log.Println("handleDel", "url.ParseQuery", err)
			fmt.Fprintln(w, errorf(err.Error()))
			return
		} else if r == nil {
			fmt.Fprintln(w, errorf("no path with id %s", id))
			return
		}
		relative = string(r)
	} else {
		relative = strings.TrimPrefix(relative, "/")
	}

	if err := del(relative); err != nil {
		log.Println("handleDel", err)
		err = errors.Unwrap(err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	b, err := json.Marshal(Base{OK: true})
	if err != nil {
		log.Println("handleDel", "json.Marshal", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}
	fmt.Fprintln(w, string(b))
}

func handleMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fmt.Fprintln(w, errorf("invalid request, expected GET got %s", r.Method))
		return
	}

	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		log.Println("handleMove", "url.ParseQuery", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	oldpath := values.Get("oldpath")
	if oldpath == "" {
		id := values.Get("id")
		if id == "" {
			fmt.Fprintln(w, errorf("missing either oldpath or id query parameter"))
			return
		}

		p, err := ccID.Get([]byte(id))
		if err != nil {
			log.Println("handleMove", "ccID.Get", err)
			fmt.Fprintln(w, errorf(err.Error()))
			return
		} else if p == nil {
			fmt.Fprintln(w, errorf("no path with id %s", id))
			return
		}
		oldpath = string(p)
	}

	newpath := values.Get("newpath")
	if newpath == "" {
		fmt.Fprintln(w, errorf("missing newpath query parameter"))
		return
	}

	if err := move(oldpath, newpath); err != nil {
		log.Println("handleMove", err)
		err = errors.Unwrap(err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	b, err := json.Marshal(Base{OK: true})
	if err != nil {
		log.Println("handleMove", "json.Marshal", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}
	fmt.Fprintln(w, string(b))
}

func handleSha256sum(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		fmt.Fprintln(w, errorf("invalid request, expected GET got %s", r.Method))
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/sha256sum")
	if path == "" {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			log.Println("handleSha256sum", "url.ParseQuery", err)
			fmt.Fprintln(w, errorf(err.Error()))
			return
		}

		id := values.Get("id")
		if id == "" {
			fmt.Fprintln(w, errorf("missing id query parameter or path"))
			return
		}

		r, err := ccID.Get([]byte(id))
		if err != nil {
			log.Println("handleSha256sum", "url.ParseQuery", err)
			fmt.Fprintln(w, errorf(err.Error()))
			return
		} else if r == nil {
			fmt.Fprintln(w, errorf("no path with id %s", id))
			return
		}
		path = string(r)
	} else {
		path = strings.TrimPrefix(path, "/")
	}

	c, err := getSha256sum(path)
	if err != nil {
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	b, err := json.Marshal(ChecksumResponse{
		Base:   Base{OK: true},
		Sha256: c,
		File:   path,
	})
	if err != nil {
		log.Println("handleSha256sum", "json.Marshal", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}
	fmt.Fprintln(w, string(b))
}

func handleGetMeta(w http.ResponseWriter, r *http.Request) {
	var (
		files []File
		errs  []error
	)

	if r.Method != http.MethodGet {
		fmt.Fprintln(w, errorf("invalid request, expected GET got %s", r.Method))
		return
	}

	err := ccID.Fold(func(id, path []byte) error {
		h, err := ccHash.Get(path)
		if err != nil {
			log.Println("handleGetMeta", "ccHash.Get", err)
			errs = append(errs, err)
			return nil
		}

		files = append(files, File{
			ID:        string(id),
			Path:      string(path),
			Sha256sum: string(h),
		})
		return nil
	})
	if err != nil {
		log.Println("handleGetMeta", "ccID.Fold", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	b, err := json.Marshal(PutResponse{
		Base:   Base{OK: len(errs) == 0},
		Files:  files,
		Errors: errs,
	})
	if err != nil {
		log.Println("handleGetMeta", "json.Marshal", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	fmt.Fprintln(w, string(b))
}

func handleSetMeta(w http.ResponseWriter, r *http.Request) {
	var files []File

	if r.Method != http.MethodPost {
		fmt.Fprintln(w, errorf("invalid request, expected POST got %s", r.Method))
		return
	}

	err := json.NewDecoder(r.Body).Decode(&files)
	if err != nil {
		log.Println("handleSetMeta", "json.Decoder.Decode", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	errs := restore(files)
	b, err := json.Marshal(PutResponse{
		Base:   Base{OK: len(errs) == 0},
		Errors: errs,
	})
	if err != nil {
		log.Println("handleSetMeta", "json.Marshal", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	fmt.Fprintln(w, string(b))
}

func createIfNotExists(path string) {
	if ok, err := exists(path); err != nil {
		log.Fatalf("could not create dir %q, reason: %v", path, err)
	} else if !ok {
		if err := os.MkdirAll(path, 0755); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	var (
		port       int
		basedir    string
		ccdir      string
		backupFile string
	)

	flag.IntVar(&port, "p", 0, "The port Adam will listen to.")
	flag.StringVar(&basedir, "d", "", "The directory Adam will use as root directory.")
	flag.StringVar(&ccdir, "c", "", "The directory Adam will use to cache the checksums.")
	flag.StringVar(&backupFile, "restore", "", "The path to the json file Adam will use to restore the caches.")
	flag.Parse()

	cfgpath := filepath.Join(Home, ".config", "adam.toml")
	cfg = parseConfig(cfgpath)

	if ccdir != "" {
		cfg.CacheDir = ccdir
	} else if cfg.CacheDir == "" {
		cfg.CacheDir = filepath.Join(Home, ".cache", "adam")
		log.Println("no cache directory specified, falling back to", cfg.CacheDir)
	}

	if backupFile != "" {
		if errs := restoreFile(backupFile); len(errs) != 0 {
			for _, e := range errs {
				fmt.Println(e)
			}
		} else {
			fmt.Println("ok")
		}
		return
	}

	if port != 0 {
		cfg.Port = fmt.Sprintf(":%d", port)
	} else if cfg.Port == "" {
		cfg.Port = ":8080"
		log.Println("no port specified, falling back to :8080")
	}

	if basedir != "" {
		cfg.BaseDir = basedir
	} else if cfg.BaseDir == "" {
		cfg.BaseDir = filepath.Join(Home, ".adam")
		log.Println("no base directory specified, falling back to", cfg.BaseDir)
	}

	go createIfNotExists(cfg.BaseDir)
	go createIfNotExists(cfg.CacheDir)

	ccHash = Cache(filepath.Join(cfg.CacheDir, "sha256sum"))
	ccID = Cache(filepath.Join(cfg.CacheDir, "ids"))

	log.Println("setting the base directory at", cfg.BaseDir)
	log.Println("adam is running...")

	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(cfg.BaseDir))))
	http.HandleFunc("/get", handleGet)
	http.HandleFunc("/put", handlePut)
	http.HandleFunc("/put/", handlePut)
	http.HandleFunc("/del", handleDel)
	http.HandleFunc("/del/", handleDel)
	http.HandleFunc("/move", handleMove)
	http.HandleFunc("/sha256sum", handleSha256sum)
	http.HandleFunc("/sha256sum/", handleSha256sum)
	http.HandleFunc("/get_meta", handleGetMeta)
	http.HandleFunc("/set_meta", handleSetMeta)

	log.Fatal(http.ListenAndServe(cfg.Port, nil))
}
