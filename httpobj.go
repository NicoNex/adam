/*
 * Adam - Adam's a Data Access Manager
 * Copyright (C) 2021 Nicolò Santamaria
 *
 * Adam is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
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

// Base is the base json returned after each request.
type Base struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// PutResponse represents the json returned after a /put call.
type PutResponse struct {
	Base
	Files  []File  `json:"files,omitempty"`
	Errors []error `json:"errors,omitempty"`
}

// ChecksumResponse represents the json returned after a /sha256sum call.
type ChecksumResponse struct {
	Base
	File   string `json:"file"`
	Sha256 string `json:"sha256sum"`
}

// File represents the json containing all the metadata of a file.
type File struct {
	Path      string `json:"path"`
	Sha256sum string `json:"sha256sum,omitempty"`
	ID        string `json:"id,omitempty"`
}

// InputFile represents the json containing a file content encoded in base64
// and its metadata.
type InputFile struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Content string `json:"content"`
}
