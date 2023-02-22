// main allows you to build go2md binary
// # go2md
//
// [![](https://img.shields.io/github/actions/workflow/status/jylitalo/go2md/test.yml?branch=main&longCache=true&label=Test&logo=github%20actions&logoColor=fff)](https://github.com/jylitalo/go2md/actions?query=workflow%3ATest)
// [![Go Reference](https://pkg.go.dev/badge/github.com/jylitalo/go2md.svg)](https://pkg.go.dev/github.com/jylitalo/go2md)
// [![Go Report Card](https://goreportcard.com/badge/github.com/jylitalo/go2md)](https://goreportcard.com/report/github.com/jylitalo/go2md)
//
// Create markdown documentation from golang code.
//
// This project was inspired by [github.com/davecheney/godoc2md](https://github.com/davecheney/godoc2md), but why I wanted to build it without dependency to golang.org/x/tools v0.0.0-20181011021141-0e57ebad1d6b
//
// Main target audience is private golang projects or anyone, who wants to keep documentation in git repo as files.
//
// ## Build binary
//
// `go build go2md.go` will produce you go2md binary.
package main
