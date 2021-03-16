package ui

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed build/*
var webfs embed.FS

func MustFS() fs.FS {
	fsn, err := fs.Sub(webfs, "build")
	if err != nil {
		log.Fatal(err)
	}

	return fsn
}
