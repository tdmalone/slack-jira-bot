// +build ignore

// Copyright 2016 govend. All rights reserved.
// Use of this source code is governed by an Apache 2.0
// license that can be found in the LICENSE file.

//
// This file generates stdpkgs.go, which contains the standard library packages.
//
// This file has been modified from its original source:
// https://github.com/golang/tools/blob/master/imports/mkindex.go
//

package main

import (
	"bytes"
	"fmt"
	"go/build"
	"go/format"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
)

var (
	pkgs = map[string][]stdpkg{}
	fset = token.NewFileSet()
)

// stdpkg represents a standard package in "stdpkgs.go".
type stdpkg struct {
	path string // full pkg import path, e.g. "net/http"
	dir  string // absolute file path to pkg directory e.g. "/usr/lib/go/src/fmt"
}

func main() {

	// start with the default context
	ctx := build.Default

	// remove the GOPATH, we only want to search packages in the GOROOT
	ctx.GOPATH = ""

	// iterate through the list of package source root directories
	for _, path := range ctx.SrcDirs() {

		// open the file
		f, err := os.Open(path)
		if err != nil {
			log.Print(err)
			continue
		}

		// gather all the child names from the directory in a single slice
		children, err := f.Readdir(-1)
		f.Close() // close the file
		if err != nil {
			log.Print(err)
			continue
		}

		// iterate through each child name
		for _, child := range children {
			if child.IsDir() { // check the child name is a directory
				load(path, child.Name()) // load the package path and name.
			}
		}
	}

	// write preliminary file data such as comments, package name, structs, etc..
	var buf bytes.Buffer
	buf.WriteString(`// Copyright 2016 govend. All rights reserved.
	// Use of this source code is governed by an Apache 2.0
	// license that can be found in the LICENSE file.

	//
	// this file is auto-generated by generate.go
	//

	`)
	buf.WriteString("package filters\n")
	buf.WriteString(`
	type stdpkg struct {
		path, dir string
	}
	`)

	keys := make([]string, len(pkgs), len(pkgs))
	idx := 0
	for key, _ := range pkgs {
		keys[idx] = key
		idx += 1
	}

	sort.Strings(keys)

	// write the dynamic list of standard packages
	fmt.Fprintf(&buf, "var stdpkgs = map[string][]stdpkg{\n")
	for _, key := range keys {
		fmt.Fprintf(&buf, "\"%s\": %#v,\n", key, pkgs[key])
	}
	fmt.Fprintf(&buf, "}")

	// transfer buffer bytes to final source
	src := buf.Bytes()

	// replace main.pkg type name with pkg
	src = bytes.Replace(src, []byte("main.stdpkg"), []byte("stdpkg"), -1)

	// replace actual GOROOT with "/go"
	src = bytes.Replace(src, []byte(ctx.GOROOT), []byte("/go"), -1)

	// add line wrapping and better formatting.
	src = bytes.Replace(src, []byte("[]stdpkg{stdpkg{"), []byte("{\n{"), -1)
	src = bytes.Replace(src, []byte(", stdpkg"), []byte(",\nstdpkg"), -1)
	src = bytes.Replace(src, []byte("stdpkg{path"), []byte("{path"), -1)
	src = bytes.Replace(src, []byte("}}, "), []byte("},\n},\n"), -1)
	src = bytes.Replace(src, []byte("true, "), []byte("true,\n"), -1)
	src = bytes.Replace(src, []byte("}}}"), []byte("},\n},\n}"), -1)

	// format all the source bytes
	src, err := format.Source(src)
	if err != nil {
		log.Fatal(err)
	}

	// write source bytes to the "stdpkgs.go" file
	if err := ioutil.WriteFile("stdpkgs.go", src, 0644); err != nil {
		log.Fatal(err)
	}
}

// load takes a path root and import path.
func load(root, importpath string) {

	// get package name
	name := path.Base(importpath)
	if name == "testdata" {
		return
	}

	// calculate the package source directory
	dir := filepath.Join(root, importpath)

	// append the package values to the package map
	pkgs[name] = append(pkgs[name], stdpkg{
		path: importpath,
		dir:  dir,
	})

	// get the package directory
	pkgDir, err := os.Open(dir)
	if err != nil {
		return
	}

	// gather all the child names from the directory in a single slice
	children, err := pkgDir.Readdir(-1)

	// close the file and check for errors
	pkgDir.Close()
	if err != nil {
		return
	}

	// iterate through each child name
	for _, child := range children {

		name := child.Name()
		if name == "" { // check that the childs names not blank
			continue
		}

		// handle special package name cases
		if c := name[0]; c == '.' || ('0' <= c && c <= '9') {
			continue
		}

		// check if the child name is a directory
		if child.IsDir() {
			load(root, filepath.Join(importpath, name)) // load package path and name
		}
	}
}
