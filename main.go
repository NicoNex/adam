package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
		log.Println("errorf", err)
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
			log.Println("saveFile", err)
			return err
		}
	} else if err != nil {
		log.Println("saveFile", err)
		return err
	}

	if err := os.WriteFile(fpath, content, 0755); err != nil {
		log.Println("saveFile", err)
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
				log.Println("handlePut", err)
				continue
			}

			fname := filepath.Base(h.Filename)
			cnt, err := io.ReadAll(tmp)
			tmp.Close()
			if err != nil {
				log.Println("handlePut", err)
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
					log.Println("handlePut", "error saving file", err)
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
			log.Println("handlePut", err)
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
		log.Println("handleDel", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	if !ok {
		fmt.Fprintln(w, errorf("unable to find path %q", path))
		return
	}

	log.Println("handleDel", "deleting path", path)
	if err := os.RemoveAll(path); err != nil {
		log.Println("handleDel", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}

	b, err := json.Marshal(Base{OK: true})
	if err != nil {
		log.Println("handleDel", err)
		fmt.Fprintln(w, errorf(err.Error()))
		return
	}
	fmt.Fprintln(w, string(b))
}

func main() {
	log.Println("adam is running...")

	cfgpath := filepath.Join(Home, ".config", "adam.toml")
	log.Println("read config file at", cfgpath)
	cfg = parseConfig(cfgpath)

	log.Println("setting the base directory at", cfg.BaseDir)
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(cfg.BaseDir))))
	http.HandleFunc("/put/", handlePut)
	http.HandleFunc("/del/", handleDel)

	log.Fatal(http.ListenAndServe(cfg.Port, nil))
}