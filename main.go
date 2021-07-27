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
		wg         sync.WaitGroup
		savedFiles FileList
		errs       ErrList
		ok         = true
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
			fpath := filepath.Join(cfg.BaseDir, fdir, fname)
			wg.Add(1)
			go func(fdir, fpath string, cnt []byte) {
				defer wg.Done()

				if err := saveFile(fpath, cnt); err != nil {
					ok = false
					errs.Append(err)
					log.Println("handlePut", "saveFile", err)
					return
				}

				path := filepath.Join(fdir, fname)

				hash, err := saveSha256sum(path, cnt)
				if err != nil {
					log.Println("handlePut", "saveSha256sum", err)
					errs.Append(err)
					ok = false
				}

				id, err := uuid.NewRandom()
				if err != nil {
					log.Println("handlePut", "uuid.NewRandom", err)
					errs.Append(err)
					savedFiles.Append(File{Path: filepath.Join(fdir, fname)})
					ok = false
					return
				}

				savedFiles.Append(File{
					Path:      path,
					ID:        id.String(),
					Sha256sum: hash,
				})

				if err := ccID.Put([]byte(id.String()), []byte(path)); err != nil {
					log.Println("handlePut", "ccID.Put", err)
					errs.Append(err)
					ok = false
				}
			}(fdir, fpath, cnt)
		}
		wg.Wait()

		b, err := json.Marshal(PutResponse{
			Base:   Base{OK: ok},
			Files:  savedFiles.Slice(),
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

	path := filepath.Join(cfg.BaseDir, relative)

	ok, err := exists(path)
	if err != nil {
		log.Println("handleDel", "exists", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	if !ok {
		fmt.Fprintln(w, errorf("unable to find path %s", path))
		return
	}

	if err := os.RemoveAll(path); err != nil {
		log.Println("handleDel", "os.RemoveAll", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	go func() {
		if err := ccHash.Del([]byte(relative)); err != nil {
			log.Println("handleDel", "ccHash.Del", err)
		}
	}()
	go func() {
		if err := ccID.Del([]byte(relative)); err != nil {
			log.Println("handleDel", "ccID.Del", err)
		}
	}()

	b, err := json.Marshal(Base{OK: true})
	if err != nil {
		log.Println("handleDel", "json.Marshal", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}
	fmt.Fprintln(w, string(b))
}

func handleMove(w http.ResponseWriter, r *http.Request) {
	var fileID string

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

		fileID = id
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

	destDir := filepath.Dir(newpath)
	if ok, err := exists(destDir); !ok && err == nil {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			log.Println("handleMove", "os.MkdirAll", err)
			fmt.Fprintln(w, errorf(err.Error()))
			return
		}
	} else if err != nil {
		log.Println("handleMove", "os.MkdirAll", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	go func(oldpath, newpath string) {
		if err := moveSha256sum(oldpath, newpath); err != nil {
			log.Println("handleMove", "moveSha256sum", err)
		}
	}(oldpath, newpath)
	go func(oldpath, newpath, fileID string) {
		if fileID == "" {
			id, err := findIDFromPath(oldpath)
			if err != nil {
				log.Println("handleMove", "getIDFromPath", err)
				return
			}
			fileID = id
		}

		if err := ccID.Put([]byte(fileID), []byte(newpath)); err != nil {
			log.Println("handleMove", "ccID.Put", err)
		}
	}(oldpath, newpath, fileID)

	oldpath = filepath.Join(cfg.BaseDir, oldpath)
	newpath = filepath.Join(cfg.BaseDir, newpath)
	if err := os.Rename(oldpath, newpath); err != nil {
		log.Println("handleMove", "os.Rename", err)
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

func createIfNotExists(path string) {
	if ok, err := exists(path); err != nil {
		log.Fatal("could not create dir %q, reason: %v", path, err)
	} else if !ok {
		if err := os.MkdirAll(path, 0755); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	var port int
	var basedir, ccdir string

	flag.IntVar(&port, "p", 0, "The port Adam will listen to.")
	flag.StringVar(&basedir, "d", "", "The directory Adam will use as root directory.")
	flag.StringVar(&ccdir, "c", "", "The directory Adam will use to cache the checksums.")
	flag.Parse()

	log.Println("adam is running...")

	cfgpath := filepath.Join(Home, ".config", "adam.toml")
	cfg = parseConfig(cfgpath)

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

	if ccdir != "" {
		cfg.CacheDir = ccdir
	} else if cfg.CacheDir == "" {
		cfg.CacheDir = filepath.Join(Home, ".cache", "adam")
		log.Println("no cache directory specified, falling back to", cfg.CacheDir)
	}

	go createIfNotExists(cfg.BaseDir)
	go createIfNotExists(cfg.CacheDir)

	ccHash = Cache(filepath.Join(cfg.CacheDir, "sha256sum"))
	ccID = Cache(filepath.Join(cfg.CacheDir, "ids"))

	log.Println("setting the base directory at", cfg.BaseDir)
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(cfg.BaseDir))))
	http.HandleFunc("/get", handleGet)
	http.HandleFunc("/put", handlePut)
	http.HandleFunc("/put/", handlePut)
	http.HandleFunc("/del", handleDel)
	http.HandleFunc("/del/", handleDel)
	http.HandleFunc("/move", handleMove)
	http.HandleFunc("/sha256sum", handleSha256sum)
	http.HandleFunc("/sha256sum/", handleSha256sum)

	log.Fatal(http.ListenAndServe(cfg.Port, nil))
}
