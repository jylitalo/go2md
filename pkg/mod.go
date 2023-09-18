package pkg

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	ErrModuleNameMissing = errors.New("failed to find module name")
	ErrGoModMissing      = errors.New("unable to find go.mod")
)

func hasGoMod(dir string) bool {
	// log.Info("Checking " + dir)
	_, err := os.Stat(dir + "/go.mod")
	return !os.IsNotExist(err)
}

func moduleName(dir string) (string, error) {
	f, err := os.Open(filepath.Clean(dir + "/go.mod"))
	if err != nil {
		err = fmt.Errorf("moduleName failed: %w", err)
		log.WithFields(log.Fields{"dir": dir}).Error(err)
		return "", err
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
	return "", fmt.Errorf("%w from %s/go.mod", ErrModuleNameMissing, dir)
}

// getPackageName assumes that each directory has one package name in golang namespace.
func getPackageName(dir string) (string, error) {
	cwd, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("getPackageName failed: %w", err)
	}
	dirs := strings.Split(cwd, "/")
	for idx := len(dirs); idx > 0; idx-- {
		root := strings.Join(dirs[0:idx], "/")
		if !hasGoMod(root) {
			continue
		}
		modName, err := moduleName(root)
		if err != nil {
			return "", err
		}
		parts := append([]string{modName}, dirs[idx:]...)
		return strings.Join(parts, "/"), nil
	}
	return "", fmt.Errorf("%w from %s or its parent dirs", ErrGoModMissing, cwd)
}

func fileExists(fname string) bool {
	_, err := os.Stat(fname)
	return err == nil || !os.IsNotExist(err)
}
