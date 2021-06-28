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
)

var (
	cfg    Config
	ccHash Cache
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
	if err := os.WriteFile(fpath, content, 0755); err != nil {
		return err
	}
	return nil
}

func saveSha256sum(fpath string) (string, error) {
	f, err := os.Open(fpath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
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

func handlePut(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
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
		savedFiles StrList
		errors     ErrList
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

			fdir := strings.TrimPrefix(r.URL.Path, "/put/")
			fpath := filepath.Join(cfg.BaseDir, fdir, fname)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := saveFile(fpath, cnt); err == nil {
					go func() {
						if _, err := saveSha256sum(fpath); err != nil {
							log.Println("handlePut", "saveSha256sum", err)
						}
					}()
					savedFiles.Append(filepath.Join(fdir, fname))
				} else {
					ok = false
					errors.Append(err)
					log.Println("handlePut", "saveFile", err)
				}
			}()
		}
		wg.Wait()

		b, err := json.Marshal(PutResponse{
			Base:   Base{OK: ok},
			Files:  savedFiles.Slice(),
			Errors: errors.Slice(),
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
	path := filepath.Join(cfg.BaseDir, strings.TrimPrefix(r.URL.Path, "/del/"))

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
		if err := ccHash.Del([]byte(path)); err != nil {
			log.Println("handleDel", "ccHash.Del", err)
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
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		log.Println("handleMove", "url.ParseQuery", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	oldpath := values.Get("oldpath")
	if oldpath == "" {
		fmt.Fprintln(w, errorf("missing oldpath query parameter"))
		return
	}
	oldpath = filepath.Join(cfg.BaseDir, oldpath)

	newpath := values.Get("newpath")
	if newpath == "" {
		fmt.Fprintln(w, errorf("missing oldpath query parameter"))
		return
	}
	newpath = filepath.Join(cfg.BaseDir, newpath)

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

	if err := os.Rename(oldpath, newpath); err != nil {
		log.Println("handleMove", "os.Rename", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	go func() {
		if err := moveSha256sum(oldpath, newpath); err != nil {
			log.Println("handleMove", "moveSha256sum", err)
		}
	}()

	b, err := json.Marshal(Base{OK: true})
	if err != nil {
		log.Println("handleMove", "json.Marshal", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}
	fmt.Fprintln(w, string(b))
}

func handleSha256sum(w http.ResponseWriter, r *http.Request) {
	relative := strings.TrimPrefix(r.URL.Path, "/sha256sum/")
	path := filepath.Join(cfg.BaseDir, relative)

	c, err := getSha256sum(path)
	if err != nil {
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	b, err := json.Marshal(ChecksumResponse{
		Base:   Base{OK: true},
		Sha256: c,
		File:   relative,
	})
	if err != nil {
		log.Println("handleSha256sum", "json.Marshal", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}
	fmt.Fprintln(w, string(b))
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

	ok, err := exists(cfg.CacheDir)
	if err != nil {
		log.Fatal(err)
	} else if !ok {
		log.Println("creating directory with path", cfg.CacheDir)
		if err := os.MkdirAll(cfg.CacheDir, 0755); err != nil {
			log.Fatal(err)
		}
	}

	ccHash = Cache(filepath.Join(cfg.CacheDir, "sha256sum"))

	log.Println("setting the base directory at", cfg.BaseDir)
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(cfg.BaseDir))))
	http.HandleFunc("/put/", handlePut)
	http.HandleFunc("/del/", handleDel)
	http.HandleFunc("/move", handleMove)
	http.HandleFunc("/sha256sum/", handleSha256sum)

	log.Fatal(http.ListenAndServe(cfg.Port, nil))
}
