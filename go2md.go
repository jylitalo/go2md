package main

import (
	_ "embed"

	"github.com/jylitalo/go2md/cmd"

	log "github.com/sirupsen/logrus"
)

//go:embed version.txt
var Version string // value from version.txt file

func main() {
	if err := cmd.NewCommand(Version).Execute(); err != nil {
		log.Fatal(err)
	}
}
