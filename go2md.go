package main

import (
	_ "embed"
	"io"
	"log/slog"
	"os"
	"slices"

	"github.com/jylitalo/go2md/cmd"
	"github.com/jylitalo/tint"
	"github.com/mattn/go-isatty"
)

//go:embed version.txt
var Version string // value from version.txt file

func execute(writer io.WriteCloser) error {
	return cmd.NewCommand(writer, Version).Execute()
}

func setupLogging(debug bool, color bool) {
	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}
	w := os.Stderr
	log := slog.New(tint.NewHandler(w, &tint.Options{
		Level:   logLevel,
		NoColor: !isatty.IsTerminal(w.Fd()) || !color,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		},
	}))
	slog.SetDefault(log)
	slog.Debug("go2md started")
}

func main() {
	setupLogging(slices.Contains(os.Args, "--debug"), !slices.Contains(os.Args, "--no-color"))
	if err := execute(os.Stdout); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
