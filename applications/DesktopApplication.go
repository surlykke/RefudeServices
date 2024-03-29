// Copyright (c) Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
package applications

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/surlykke/RefudeServices/lib/link"
	"github.com/surlykke/RefudeServices/lib/requests"
	"github.com/surlykke/RefudeServices/lib/resource"
	"github.com/surlykke/RefudeServices/lib/respond"

	"github.com/surlykke/RefudeServices/lib/xdg"
)

type DesktopApplication struct {
	resource.BaseResource
	Type            string
	Version         string `json:",omitempty"`
	GenericName     string `json:",omitempty"`
	NoDisplay       bool
	Hidden          bool
	OnlyShowIn      []string
	NotShowIn       []string
	DbusActivatable bool   `json:",omitempty"`
	TryExec         string `json:",omitempty"`
	Exec            string `json:",omitempty"`
	WorkingDir      string //`json:",omitempty"`
	Terminal        bool
	Categories      []string
	Implements      []string
	StartupNotify   bool
	StartupWmClass  string `json:",omitempty"`
	Url             string `json:",omitempty"`
	DesktopActions  []DesktopAction
	DesktopId       string
	Mimetypes       []string
	DesktopFile     string
}

func (d *DesktopApplication) Actions() link.ActionList {
	var ll = link.ActionList{link.MkAction("", "Launch", d.IconUrl)}
	for _, da := range d.DesktopActions {
		ll = append(ll, link.MkAction(da.id, da.Name, da.IconUrl))
	}
	return ll
}

func (d *DesktopApplication) RelevantForSearch() bool {
	return !d.NoDisplay
}

func (d *DesktopApplication) Run(arg string) error {
	return run(d.Exec, arg, d.Terminal)
}

type DesktopAction struct {
	id      string
	Name    string
	Exec    string
	IconUrl link.Href
}

func (d *DesktopApplication) DoPost(w http.ResponseWriter, r *http.Request) {
	var exec string
	var terminal bool
	var action = requests.GetSingleQueryParameter(r, "action", "")
	if action == "" {
		exec, terminal = d.Exec, d.Terminal
	} else {
		for _, da := range d.DesktopActions {
			if action == da.id {
				exec = da.Exec
			}
		}
	}
	if exec != "" {
		if err := run(exec, "", terminal); err != nil {
			respond.ServerError(w, err)
		} else {
			respond.Accepted(w)
		}
	} else {
		respond.NotFound(w)
	}
}

var Applications = resource.MakeCollection[*DesktopApplication]("/application/")

func GetAppsIds(mimetypeId string) []string {
	if res, ok := Mimetypes.Get(mimetypeId); ok {
		return res.Applications
	} else {
		return []string{}
	}
}

func GetApps(appIds ...string) []*DesktopApplication {
	var apps = make([]*DesktopApplication, 0, len(appIds))
	for _, appId := range appIds {
		if res, ok := Applications.Get(appId); ok {
			apps = append(apps, res)
		}
	}
	return apps
}

func OpenFile(appId, path string) (bool, error) {
	if appId == "" {
		xdg.RunCmd("xdg-open", path)
		return true, nil
	} else if res, ok := Applications.Get(appId); ok {
		return true, res.Run(path)
	} else {
		return false, nil
	}
}

func GetIconUrl(appId string) link.Href {
	if app, ok := Applications.Get(appId); ok {
		return app.IconUrl
	} else {
		return ""
	}
}

var argPlaceholders = regexp.MustCompile("%[uUfF]")

func run(exec string, arg string, inTerminal bool) error {
	var argv = strings.Fields(exec)
	for i := 0; i < len(argv); i++ {
		argv[i] = argPlaceholders.ReplaceAllString(argv[i], arg)
	}

	// Get rid of empty arguments
	var left = 0
	for i := 0; i < len(argv); i++ {
		if len(strings.TrimSpace(argv[i])) > 0 {
			argv[left] = strings.TrimSpace(argv[i])
			left++
		}
	}
	argv = argv[0:left]

	if inTerminal {
		var terminal, ok = os.LookupEnv("TERMINAL")
		if !ok {
			return fmt.Errorf("trying to run %s in terminal, but env variable TERMINAL not set", exec)
		}
		argv = append([]string{terminal, "-e"}, argv...)
	}

	return xdg.RunCmd(argv...)
}
