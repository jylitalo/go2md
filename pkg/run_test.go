package pkg

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestRun generates markdown from pkg and compares it against README.md file.
func TestRun(t *testing.T) {
	t.Run("validate output", func(t *testing.T) {
		var b bytes.Buffer

		versionBytes, err := os.ReadFile("../version.txt")
		if err != nil {
			t.Error("failed to read version file")
		}
		version := strings.TrimSpace(string(versionBytes))
		readme, err := os.ReadFile("README.md")
		if err != nil {
			t.Error("failed to read README.md")
		}

		err = Run(&b, version)
		if err != nil {
			t.Error("Run() returned err: " + err.Error())
		}
		expected := strings.TrimSpace(string(readme))
		received := strings.TrimSpace(b.String())
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
}
