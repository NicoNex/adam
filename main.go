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

var cfg Config

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
		log.Println("saveFile", "creating directory with path", dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Println("saveFile", "os.MkdirAll", err)
			return err
		}
	} else if err != nil {
		log.Println("saveFile", "exists", err)
		return err
	}

	if err := os.WriteFile(fpath, content, 0755); err != nil {
		log.Println("saveFile", "os.WriteFile", err)
		return err
	}
	return nil
}

func handlePut(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fmt.Fprintln(w, errorf("invalid request, expected \"POST\" got %q", r.Method))
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

	var mu sync.Mutex
	var wg sync.WaitGroup
	var savedFiles []string
	var ok = true

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
			log.Println("handlePut", "saving file with path", fpath)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := saveFile(fpath, cnt); err != nil {
					ok = false
					log.Println("handlePut", "saveFile", err)
				} else {
					mu.Lock()
					savedFiles = append(savedFiles, fpath)
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		res := PutResponse{Base: Base{OK: ok}, Files: savedFiles}
		b, err := json.Marshal(res)
		if err != nil {
			log.Println("handlePut", "json.Marshal", err)
			fmt.Fprintln(w, errorf(err.Error()))
			return
		}
		fmt.Println(savedFiles)
		fmt.Fprintln(w, string(b))
	}
}

func handleDel(w http.ResponseWriter, r *http.Request) {
	var path = filepath.Join(cfg.BaseDir, strings.TrimPrefix(r.URL.Path, "/del/"))

	ok, err := exists(path)
	if err != nil {
		log.Println("handleDel", "exists", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	if !ok {
		fmt.Fprintln(w, errorf("unable to find path %q", path))
		return
	}

	log.Println("handleDel", "deleting path", path)
	if err := os.RemoveAll(path); err != nil {
		log.Println("handleDel", "os.RemoveAll", err)
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
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		log.Println("handleMove", "url.ParseQuery", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	source := values.Get("source")
	if source == "" {
		fmt.Fprintln(w, errorf("no source path provided"))
		return
	}
	source = filepath.Join(cfg.BaseDir, source)

	dest := values.Get("dest")
	if dest == "" {
		fmt.Fprintln(w, errorf("no dest path provided"))
		return
	}
	dest = filepath.Join(cfg.BaseDir, dest)

	if err := os.Rename(source, dest); err != nil {
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

func main() {
	var port int
	var basedir string

	flag.IntVar(&port, "p", 0, "The port Adam will listen to.")
	flag.StringVar(&basedir, "d", "", "The directory Adam will use as root directory.")
	flag.Parse()

	log.Println("adam is running...")

	cfgpath := filepath.Join(Home, ".config", "adam.toml")
	log.Println("read config file at", cfgpath)
	cfg = parseConfig(cfgpath)

	if port != 0 {
		cfg.Port = fmt.Sprintf(":%d", port)
	}
	if basedir != "" {
		cfg.BaseDir = basedir
	}

	log.Println("setting the base directory at", cfg.BaseDir)
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(cfg.BaseDir))))
	http.HandleFunc("/put/", handlePut)
	http.HandleFunc("/del/", handleDel)
	http.HandleFunc("/move", handleMove)

	log.Fatal(http.ListenAndServe(cfg.Port, nil))
}
