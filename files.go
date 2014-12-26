package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
)

// files keeps track of all files golem uses.
type files struct {
	configDir string
	cacheDir  string
	cookies   string
	rc        string
}

// newFiles initializes the files golem uses.
func (g *golem) newFiles() (*files, error) {
	home := os.Getenv("HOME")
	if home == "" {
		user, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("Failed to find $HOME!")
		}
		home = user.HomeDir
	}

	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(home, ".config")
	}
	configDir = filepath.Join(configDir, "golem", g.profile)
	err := os.MkdirAll(configDir, 0700)
	if err != nil {
		return nil, err
	}

	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = filepath.Join(home, ".cache")
	}
	cacheDir = filepath.Join(cacheDir, "golem", g.profile)
	err = os.MkdirAll(cacheDir, 0700)
	if err != nil {
		return nil, err
	}

	cookies := filepath.Join(configDir, "cookies")

	rc := filepath.Join(configDir, "golemrc")
	// If the rc file does not exist, we create it with defaultRc as its
	// content.
	_, err = os.Stat(rc)
	if err != nil && os.IsNotExist(err) {
		err = ioutil.WriteFile(rc, []byte(defaultRc), 0600)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &files{configDir, cacheDir, cookies, rc}, nil
}

// readRC reades the RC file.
func (fs *files) readRC() (string, error) {
	data, err := ioutil.ReadFile(fs.rc)
	return string(data), err
}
