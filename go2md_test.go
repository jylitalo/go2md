package main

import (
	"os"
	"strings"
	"testing"
)

// TestRun generates markdown from pkg and compares it against README.md file.
func TestMain(t *testing.T) {
	t.Run("validate output", func(t *testing.T) {
		readme, err := os.ReadFile("README.md")
		if err != nil {
			t.Error("failed to read README.md")
		}
		os.Args = []string{"go2md", "--output=README2.md"}
		main()
		readme2, err := os.ReadFile("README2.md")
		if err != nil {
			t.Error("failed to read README2.md")
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
}
