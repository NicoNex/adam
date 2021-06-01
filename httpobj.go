package main

type Base struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type PutResponse struct {
	Base
	Files []string `json:"files,omitempty"`
}
