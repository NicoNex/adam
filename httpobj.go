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

type Base struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type PutResponse struct {
	Base
	Files []string `json:"files,omitempty"`
}

type ChecksumResponse struct {
	Base
	File   string `json:"file"`
	Sha256 string `json:"sha256sum"`
}
