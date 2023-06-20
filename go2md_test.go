package main

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jylitalo/go2md/pkg"
)

// TestRun generates markdown from pkg and compares it against README.md file.
func TestMain(t *testing.T) {
	t.Run("validate output", func(t *testing.T) {
		tmp := "README2.md"
		readme, err := os.ReadFile("README.md")
		if err != nil {
			t.Error("failed to read README.md")
		}
		os.Args = []string{"go2md", "--output=" + tmp}
		defer func() {
			_ = os.Remove(tmp)
		}()
		main()
		readme2, err := os.ReadFile(tmp)
		if err != nil {
			t.Error("failed to read " + tmp)
		}
		expected := strings.TrimSpace(string(readme))
		received := strings.TrimSpace(string(readme2))
		if expected != received {
			t.Error("outputs don't match")
			expLines := strings.Split(expected, "\n")
			recvLines := strings.Split(received, "\n")
			if len(expLines) != len(recvLines) {
				t.Logf("Number of lines (expected %d vs. received %d", len(expLines), len(recvLines))
			}
			commonLines := len(expLines)
			if commonLines < len(recvLines) {
				commonLines = len(recvLines)
			}

			for i := 0; i < commonLines; i++ {
				if expLines[i] != recvLines[i] {
					t.Logf("Line #%d: %s vs. %s", i, expLines[i], recvLines[i])
				}
			}
		}
	})
	t.Run("--ignore-main without --recursive", func(t *testing.T) {
		finfo, err := os.Stat("README.md")
		if err != nil {
			t.Error(err)
		}
		mtime := finfo.ModTime()
		os.Args = []string{"go2md", "--output=README.md", "--ignore-main"}
		if err = execute(os.Stdout); err == nil && errors.Is(err, pkg.ErrNoPackageFound) {
			t.Errorf("execute return %v", err)
		}
		finfo, err = os.Stat("README.md")
		if err != nil {
			t.Error(err)
		}
		if mtime != finfo.ModTime() {
			t.Errorf("modTime on README.md was changed (%v vs. %v)", mtime, finfo.ModTime())
		}
	})
	t.Run("--ignore-main with --recursive", func(t *testing.T) {
		finfo, err := os.Stat("README.md")
		if err != nil {
			t.Error(err)
		}
		mtime := finfo.ModTime()
		os.Args = []string{"go2md", "--output=README.md", "--ignore-main", "--recursive"}
		if err = execute(os.Stdout); err != nil {
			t.Errorf("execute return %v", err)
		}
		finfo, err = os.Stat("README.md")
		if err != nil {
			t.Error(err)
		}
		if mtime != finfo.ModTime() {
			t.Errorf("modTime on README.md was changed (%v vs. %v)", mtime, finfo.ModTime())
		}
	})
}
