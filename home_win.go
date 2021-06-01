// +build windows

package main

import "os"

var Home = os.Getenv("UserProfile")
