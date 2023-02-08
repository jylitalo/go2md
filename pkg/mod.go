package pkg

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

func hasGoMod(dir string) bool {
	// log.Info("Checking " + dir)
	_, err := os.Stat(dir + "/go.mod")
	return !os.IsNotExist(err)
}

func moduleName(dir string) (string, error) {
	f, err := os.Open(dir + "/go.mod")
	if err != nil {
		log.WithFields(log.Fields{"dir": dir, "err": err}).Error("Failed to open go.mod")
		return "", errors.New("failed to open go.mod")
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	scan.Split(bufio.ScanLines)
	for scan.Scan() {
		line := scan.Text()
		if strings.HasPrefix(line, "module ") {
			words := strings.Split(line, " ")
			if len(words) > 1 {
				return words[1], nil
			}
			log.WithFields(log.Fields{"line": line}).Error("module name missing from line")
		}
	}
	return "", errors.New("failed to find module name from go.mod")
}

func getPackageName(dir string) (string, error) {
	cwd, err := filepath.Abs(dir)
	if err != nil {
		return "", errors.New("unable to determine absolute path")
	}
	dirs := strings.Split(cwd, "/")
	for join := len(dirs); join > 0; join-- {
		current := strings.Join(dirs[0:join], "/")
		if hasGoMod(current) {
			mod, err := moduleName(current)
			if err != nil {
				return "", err
			}
			parts := append([]string{mod}, dirs[join:]...)
			return strings.Join(parts, "/"), nil
		}
	}
	return "", errors.New("unable to find go.mod with module name")
}

func fileExists(fname string) bool {
	_, err := os.Stat(fname)
	return err == nil || !os.IsNotExist(err)
}
