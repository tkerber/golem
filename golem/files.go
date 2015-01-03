package golem

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
)

// files keeps track of all files golem uses.
type files struct {
	configDir   string
	cacheDir    string
	cookies     string
	rc          string
	quickmarks  string
	downloadDir string
}

var configFiles = []string{
	"golemrc",
	"quickmarks",
}

// newFiles initializes the files golem uses.
func (g *Golem) newFiles() (*files, error) {
	home := os.Getenv("HOME")
	if home == "" {
		user, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("Failed to find $HOME!")
		}
		home = user.HomeDir
	}

	downloads := filepath.Join(home, "Downloads")
	stat, err := os.Stat(downloads)
	if err != nil || !stat.IsDir() {
		log.Printf("Failed to stat download dir, falling back to $HOME.")
		downloads = home
	}

	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(home, ".config")
	}
	configDir = filepath.Join(configDir, "golem", g.profile)
	err = os.MkdirAll(configDir, 0700)
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

	configFiles, err := initConfigFiles(configDir)
	if err != nil {
		return nil, err
	}

	return &files{
		configDir,
		cacheDir,
		cookies,
		configFiles[0],
		configFiles[1],
		downloads}, nil
}

// rcFiles returns a list of all files golem should use as rc files.
func (fs *files) rcFiles() []string {
	return []string{fs.rc, fs.quickmarks}
}

// initConfigFiles ensures all config files exist in the specified config
// dir.
func initConfigFiles(configDir string) ([]string, error) {
	locations := make([]string, len(configFiles))
	for i, file := range configFiles {
		locations[i] = filepath.Join(configDir, file)
		// If the config file does not exist, we create it with the default
		// content.
		_, err := os.Stat(locations[i])
		if err != nil && os.IsNotExist(err) {
			defaultCont, err := Asset(file)
			if err != nil {
				return nil, err
			}
			err = ioutil.WriteFile(locations[i], defaultCont, 0600)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
	}
	return locations, nil
}