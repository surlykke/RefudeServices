// Copyright (c) 2017 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package applications

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/surlykke/RefudeServices/lib/respond"
	"github.com/surlykke/RefudeServices/lib/searchutils"

	"github.com/surlykke/RefudeServices/lib/xdg"
)

type DesktopApplication struct {
	Type            string
	Version         string `json:",omitempty"`
	Name            string
	GenericName     string `json:",omitempty"`
	NoDisplay       bool
	Comment         string `json:",omitempty"`
	IconName        string `json:",omitempty"`
	Hidden          bool
	OnlyShowIn      []string
	NotShowIn       []string
	DbusActivatable bool   `json:",omitempty"`
	TryExec         string `json:",omitempty"`
	Exec            string `json:",omitempty"`
	Path            string `json:",omitempty"`
	Terminal        bool
	Categories      []string
	Implements      []string
	Keywords        []string
	StartupNotify   bool
	StartupWmClass  string `json:",omitempty"`
	Url             string `json:",omitempty"`
	DesktopActions  map[string]*DesktopAction
	Id              string
	Mimetypes       []string
}

func (d *DesktopApplication) ToStandardFormat() *respond.StandardFormat {
	var self = appSelf(d.Id)
	var otherActions string
	if len(d.DesktopActions) > 0 {
		otherActions = otherActionsPath(d.Id)
	}
	return &respond.StandardFormat{
		Self:         self,
		OnPost:       "Launch",
		OtherActions: otherActions,
		Type:         "application",
		Title:        d.Name,
		Comment:      d.Comment,
		IconName:     d.IconName,
		Data:         d,
		NoDisplay:    d.NoDisplay,
	}
}

func (d *DesktopApplication) collectActions(term string) respond.StandardFormatList {
	var sfl = make(respond.StandardFormatList, 0, len(d.DesktopActions))
	for _, act := range d.DesktopActions {
		if rank := searchutils.SimpleRank(act.Name, "", term); rank > -1 {
			sfl = append(sfl, act.ToStandardFormat().Ranked(rank))
		}
	}
	return sfl.SortByRank()
}

type DesktopAction struct {
	self     string
	Name     string
	Exec     string
	IconName string
}

func (da *DesktopAction) ToStandardFormat() *respond.StandardFormat {
	return &respond.StandardFormat{
		Self:     da.self,
		OnPost:   "Launch",
		Type:     "applicationaction",
		Title:    da.Name,
		IconName: da.IconName,
		Data:     da,
	}
}

func OpenFile(path string, mimetypeId string) error {
	var c = collectionStore.Load().(collection)
	if mt, ok := c.mimetypes[mimetypeId]; ok {
		if mt.DefaultApp != "" {
			if app, ok := c.applications[mt.DefaultApp]; ok {
				return LaunchWithArg(app.Exec, path, app.Terminal)
			}
		}
	}

	return xdg.RunCmd("xdg-open", path)
}

func Launch(exec string, inTerminal bool) {
	LaunchWithArgs(exec, []string{}, inTerminal)
}

var argPlaceholders = regexp.MustCompile("%[uUfF]")

func LaunchWithArg(exec string, arg string, inTerminal bool) error {
	return LaunchWithArgs(exec, []string{arg}, inTerminal)
}

func LaunchWithArgs(exec string, args []string, inTerminal bool) error {
	var argv = strings.Fields(argPlaceholders.ReplaceAllString(exec, strings.Join(args, " ")))

	if inTerminal {
		var terminal, ok = os.LookupEnv("TERMINAL")
		if !ok {
			return fmt.Errorf("Trying to run %s in terminal, but env variable TERMINAL not set", exec)
		}
		argv = append([]string{terminal, "-e"}, argv...)
	}

	return xdg.RunCmd(argv...)
}
