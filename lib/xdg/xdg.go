// Copyright (c) 2017 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package xdg

import (
	"os"

	"github.com/surlykke/RefudeServices/lib/utils"
)

var Home string
var ConfigHome string
var ConfigDirs []string
var CacheHome string
var DataHome string
var DataDirs []string
var RuntimeDir string
var CurrentDesktop []string

func init() {
	Home = os.Getenv("HOME")
	ConfigHome = notEmptyOr(os.Getenv("XDG_CONFIG_HOME"), Home+"/.config")
	ConfigDirs = utils.Split(notEmptyOr(os.Getenv("XDG_CONFIG_DIRS"), "/etc/xdg"), ":")
	CacheHome = notEmptyOr(os.Getenv("XDG_CACHE_HOME"), Home+"/.cache")
	DataHome = notEmptyOr(os.Getenv("XDG_DATA_HOME"), Home+"/.local/share")
	DataDirs = utils.Split(notEmptyOr(os.Getenv("XDG_DATA_DIRS"), "/usr/share:/usr/local/share"), ":")
	DataDirs = utils.Remove(DataDirs, DataHome)
	RuntimeDir = notEmptyOr(os.Getenv("XDG_RUNTIME_DIR"), "/tmp")
	CurrentDesktop = utils.Split(notEmptyOr(os.Getenv("XDG_CURRENT_DESKTOP"), ""), ":")
}

func notEmptyOr(primary string, secondary string) string {
	if primary != "" {
		return primary
	} else {
		return secondary
	}
}
