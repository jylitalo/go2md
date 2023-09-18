package main

import (
	_ "embed"
	"io"
	"os"

	"github.com/jylitalo/go2md/cmd"

	log "github.com/sirupsen/logrus"
)

//go:embed version.txt
var Version string // value from version.txt file

func execute(writer io.WriteCloser) error {
	return cmd.NewCommand(writer, Version).Execute()
}

func main() {
	if err := execute(os.Stdout); err != nil {
		log.Fatal(err)
	}
}
