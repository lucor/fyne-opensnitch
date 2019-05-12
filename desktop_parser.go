// This file contains functions to extract process info like and name and icon
// from Linux desktop files.
// This is a porting of the Python OpenSnitch UI python desktop parser:
// https://github.com/evilsocket/opensnitch/blob/ec6ecea/ui/opensnitch/desktop_parser.py

package main

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/evilsocket/opensnitch/daemon/log"
)

var desktopApps map[string]desktopApp

type desktopApp struct {
	Name string
	Exec string
	Icon string
}

func init() {
	// fixes contains some adjustement for the desktopApp mappings
	fixes := map[string]string{
		"/opt/google/chrome/google-chrome": "/opt/google/chrome/chrome",
		"/usr/lib/firefox/firefox.sh":      "/usr/lib/firefox/firefox",
		"/usr/bin/pidgin":                  "/usr/bin/pidgin.orig",
	}

	desktopApps = make(map[string]desktopApp)
	for _, dp := range desktopPaths() {
		desktopApp, err := parseDesktopFile(dp)
		if err != nil {
			log.Warning("Could not fetch icon for %q. Reason: %s", dp, err)
			continue
		}

		fix, ok := fixes[desktopApp.Exec]
		if ok {
			desktopApp.Exec = fix
		}

		desktopApps[desktopApp.Exec] = desktopApp
	}
}

// desktopPaths returns the paths containing desktop files
func desktopPaths() []string {
	var desktopPaths []string
	val, ok := os.LookupEnv("XDG_DATA_DIRS")
	if !ok {
		val = "/usr/share"
	}
	for _, v := range strings.Split(val, ":") {
		pattern := filepath.Join(v, "applications", "*.desktop")
		founds, err := filepath.Glob(pattern)
		if founds == nil || err != nil {
			continue
		}
		desktopPaths = append(desktopPaths, founds...)
	}

	return desktopPaths
}

// parseDesktopFile returns a desktopApp from a .desktop file
func parseDesktopFile(desktopFile string) (desktopApp, error) {
	var desktopEntrySection bool
	var da desktopApp

	b, err := ioutil.ReadFile(desktopFile)
	if err != nil {
		return da, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "[") {
			desktopEntrySection = false
			if strings.Contains(line, "Desktop Entry") {
				desktopEntrySection = true
			}
			continue
		}
		if desktopEntrySection == false {
			continue
		}

		if strings.HasPrefix(line, "Name=") {
			da.Name = strings.TrimPrefix(line, "Name=")
			continue
		}

		if strings.HasPrefix(line, "Exec=") {
			execFile := strings.TrimPrefix(line, "Exec=")
			execFile = parseExec(execFile)
			execFile, err := resolvePath(execFile)
			if err != nil {
				return da, err
			}
			da.Exec = execFile
		}

		if strings.HasPrefix(line, "Icon=") {
			icon := strings.TrimPrefix(line, "Icon=")
			if icon == "" {
				da.Icon = "default"
				continue
			}
			if filepath.IsAbs(icon) {
				da.Icon = icon
				continue
			}
			icon = filepath.Join("/usr/share/icons/hicolor/48x48/apps", icon) + ".png"
			icon, err = resolvePath(icon)
			if err != nil {
				da.Icon = "default"
				continue
			}
			da.Icon = icon
		}
	}
	if err := scanner.Err(); err != nil {
		return da, err
	}
	return da, nil
}

// parseExec parse the Exec entry into the desktop file to return the executable
func parseExec(execFile string) string {
	// remove stuff like %U
	reRemoveParams := regexp.MustCompile(`%[a-zA-Z]+`)
	//remove 'env .... command'
	reEnv := regexp.MustCompile(`^env\s+[^\s]+\s`)
	e := reRemoveParams.ReplaceAllString(execFile, "")
	e = reEnv.ReplaceAllString(e, "")
	// remove quotes
	strings.ReplaceAll(e, "'", "")

	e = strings.TrimSpace(e)
	parts := strings.Split(e, " ")

	execFile, err := exec.LookPath(parts[0])
	if err != nil {
		return ""
	}
	return execFile
}

// resolvePath returns the absolute path resolving the symlink, if any
func resolvePath(file string) (string, error) {
	file, err := filepath.EvalSymlinks(file)
	if err != nil {
		return file, err
	}

	if filepath.IsAbs(file) {
		return file, nil
	}

	return filepath.Abs(file)
}
