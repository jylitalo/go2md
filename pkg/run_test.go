package pkg

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
func TestRun(t *testing.T) {
	t.Run("validate output (RunDirTree with includeMain=false)", func(t *testing.T) {
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

		err = RunDirTree(OutputSettings{Default: &wc, Directory: "."}, version, false)
		if err != nil {
			t.Error("run() returned err: " + err.Error())
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
				}
			}
		}
	})
	t.Run("validate output (RunDirectory includeMain=true)", func(t *testing.T) {
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

		err = RunDirectory(OutputSettings{Default: &wc, Directory: "."}, version, true)
		if err != nil {
			t.Error("run() returned err: " + err.Error())
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
				}
			}
		}
	})
}
