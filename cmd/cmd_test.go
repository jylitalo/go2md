package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

type writeCloser struct {
	b bytes.Buffer
}

func (wc *writeCloser) Write(p []byte) (int, error) {
	n, err := wc.b.Write(p)
	// log.Printf("wc.Write returns n=%d, err=%v\n", n, err)
	return n, err
}

func (wc *writeCloser) String() string {
	return wc.b.String()
}

func (wc *writeCloser) Close() error {
	return nil
}

// TestRun generates markdown from pkg and compares it against README.md file.
func TestNewCommand(t *testing.T) {
	t.Run("version", func(t *testing.T) {
		var wc writeCloser

		versionBytes, err := os.ReadFile("../version.txt")
		if err != nil {
			t.Errorf("failed to read version.txt: %v", err)
		}
		version := strings.TrimSpace(string(versionBytes))
		cmd := NewCommand(&wc, version)
		cmd.SetArgs([]string{"--version"})
		if err = cmd.Execute(); err != nil {
			t.Error("Run() returned err: " + err.Error())
		}
		received := strings.TrimSpace(wc.String())
		if received != "go2md "+version {
			t.Error(received + " != go2md v" + version)
		}
	})

	t.Run("ignore main without recursive", func(t *testing.T) {
		var wc writeCloser

		versionBytes, err := os.ReadFile("../version.txt")
		if err != nil {
			t.Error("failed to read version file")
		}
		version := strings.TrimSpace(string(versionBytes))
		finfo, err := os.Stat("README.md")
		if err != nil {
			t.Error(err)
		}
		mtime := finfo.ModTime()
		cmd := NewCommand(&wc, version)
		cmd.SetArgs([]string{"--output=README.md", "--ignore-main"})
		if err = cmd.Execute(); err != nil {
			t.Errorf("Run() returned err: %v", err)
		}
		finfo, err = os.Stat("README.md")
		if err != nil {
			t.Error(err)
		}
		if mtime == finfo.ModTime() {
			t.Errorf("modTime on README.md should have changed (%v vs. %v)", mtime, finfo.ModTime())
		}
	})

	t.Run("ignore main with recursive", func(t *testing.T) {
		var wc writeCloser

		versionBytes, err := os.ReadFile("../version.txt")
		if err != nil {
			t.Error("failed to read version file")
		}
		version := strings.TrimSpace(string(versionBytes))
		finfo, err := os.Stat("README.md")
		if err != nil {
			t.Error(err)
		}
		mtime := finfo.ModTime()
		cmd := NewCommand(&wc, version)
		cmd.SetArgs([]string{"--output=README.md", "--ignore-main", "--recursive"})
		if err = cmd.Execute(); err != nil {
			t.Errorf("Run() returned err: %v", err)
		}
		finfo, err = os.Stat("README.md")
		if err != nil {
			t.Error(err)
		}
		if mtime == finfo.ModTime() {
			t.Errorf("modTime on README.md should have changed (%v vs. %v)", mtime, finfo.ModTime())
		}
	})

	t.Run("validate output", func(t *testing.T) {
		var wc writeCloser

		versionBytes, err := os.ReadFile("../version.txt")
		if err != nil {
			t.Error("failed to read version file")
		}
		version := strings.TrimSpace(string(versionBytes))
		readme, err := os.ReadFile("README.md")
		if err != nil {
			t.Error("failed to read README.md")
		}
		cmd := NewCommand(&wc, version)
		if err = cmd.Execute(); err != nil {
			t.Error("Run() returned err: " + err.Error())
		}
		expected := strings.TrimSpace(string(readme))
		received := strings.TrimSpace(wc.String())
		if expected != received {
			t.Error("outputs don't match")
			expLines := strings.Split(expected, "\n")
			recvLines := strings.Split(received, "\n")
			if len(expLines) != len(recvLines) {
				t.Logf("Number of lines (expected %d vs. received %d)", len(expLines), len(recvLines))
			}
			commonLines := len(expLines)
			if commonLines > len(recvLines) {
				commonLines = len(recvLines)
			}
			for i := 0; i < commonLines; i++ {
				if expLines[i] != recvLines[i] {
					t.Logf("Line #%d: %s vs. %s", i, expLines[i], recvLines[i])
					t.Log("Received as whole:\n" + received)
					t.Logf("%#v", wc)
				}
			}
		}
	})
}
