// Copyright (c) Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
package applications

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/surlykke/RefudeServices/lib/link"
	"github.com/surlykke/RefudeServices/lib/log"
	"github.com/surlykke/RefudeServices/lib/slice"
	"github.com/surlykke/RefudeServices/lib/xdg"
)

func Collect() {
	var mimetypes = CollectMimeTypes()

	// TODO ----- move to CollectMimeTypes ----
	// Add aliases as mimetypes
	for _, mt := range mimetypes {
		for _, alias := range aliasTypes(mt) {
			if _, ok := mimetypes[alias.Id]; !ok {
				mimetypes[alias.Id] = alias
			}
		}
	}
	// ----
	var apps = make(map[string]*DesktopApplication)
	var defaultApps = make(map[string][]string)

	for i := len(xdg.DataDirs) - 1; i >= 0; i-- {
		var dir = xdg.DataDirs[i]
		collectApplications(dir+"/applications", apps)
		readMimeappsList(dir+"/applications/mimeapps.list", apps, defaultApps)
	}

	collectApplications(xdg.DataHome+"/applications", apps)

	for _, dir := range append(xdg.ConfigDirs, xdg.ConfigHome) {
		readMimeappsList(dir+"/mimeapps.list", apps, defaultApps)
	}

	for mimetypeId, defaultAppIds := range defaultApps {
		if mimetype, ok := mimetypes[mimetypeId]; ok {
			mimetype.Applications = append(mimetype.Applications, defaultAppIds...)
		}
	}
	for appId, app := range apps {
		for _, mimetypeId := range app.Mimetypes {
			if mimetype, ok := mimetypes[mimetypeId]; ok {
				mimetype.Applications = slice.AppendIfNotThere(mimetype.Applications, appId)
			}
		}
	}

	var appResources = make([]*DesktopApplication, 0, len(apps))
	for _, app := range apps {
		appResources = append(appResources, app)
	}

	var mimetypeResources = make([]*Mimetype, 0, len(mimetypes))
	for _, mt := range mimetypes {
		mimetypeResources = append(mimetypeResources, mt)
	}

	Applications.ReplaceWith(appResources)
	Mimetypes.ReplaceWith(mimetypeResources)
}

func aliasTypes(mt *Mimetype) []*Mimetype {
	var result = make([]*Mimetype, 0, len(mt.Aliases))
	for _, id := range mt.Aliases {
		var copy = *mt
		copy.Id = id
		copy.Aliases = []string{}
		result = append(result, &copy)
	}

	return result
}

func CollectMimeTypes() map[string]*Mimetype {
	res := make(map[string]*Mimetype)

	for id, comment := range schemeHandlers {
		var mimetype, err = MakeMimetype(id)
		if err != nil {
			log.Warn("Problem making mimetype", id)
		} else {
			mimetype.Comment = comment
			res[id] = mimetype
		}
	}

	xmlCollector := struct {
		XMLName   xml.Name `xml:"mime-info"`
		MimeTypes []struct {
			Type    string `xml:"type,attr"`
			Comment []struct {
				Lang string `xml:"lang,attr"`
				Text string `xml:",chardata"`
			} `xml:"comment"`
			Acronym []struct {
				Lang string `xml:"lang,attr"`
				Text string `xml:",chardata"`
			} `xml:"acronym"`
			ExpandedAcronym []struct {
				Lang string `xml:"lang,attr"`
				Text string `xml:",chardata"`
			} `xml:"expanded-acronym"`
			Alias []struct {
				Type string `xml:"type,attr"`
			} `xml:"alias"`
			Glob []struct {
				Pattern string `xml:"pattern,attr"`
			} `xml:"glob"`
			SubClassOf []struct {
				Type string `xml:"type,attr"`
			} `xml:"sub-class-of"`
			Icon struct {
				Name string `xml:"name,attr"`
			} `xml:"icon"`
			GenericIcon struct {
				Name string `xml:"name,attr"`
			} `xml:"generic-icon"`
		} `xml:"mime-type"`
	}{}

	xmlInput, err := os.ReadFile(freedesktopOrgXml)
	if err != nil {
		log.Warn("Unable to open ", freedesktopOrgXml, ": ", err)
	}
	parseErr := xml.Unmarshal(xmlInput, &xmlCollector)
	if parseErr != nil {
		log.Warn("Error parsing: ", parseErr)
	}

	for _, tmp := range xmlCollector.MimeTypes {
		if mimeType, err := MakeMimetype(tmp.Type); err != nil {
			log.Warn(err)
		} else {
			for _, tmpComment := range tmp.Comment {
				if xdg.LocaleMatch(tmpComment.Lang) || (tmpComment.Lang == "" && mimeType.Comment == "") {
					mimeType.Comment = tmpComment.Text
				}
			}

			for _, tmpAcronym := range tmp.Acronym {
				if xdg.LocaleMatch(tmpAcronym.Lang) || (tmpAcronym.Lang == "" && mimeType.Acronym == "") {
					mimeType.Acronym = tmpAcronym.Text
				}
			}

			for _, tmpExpandedAcronym := range tmp.ExpandedAcronym {
				if xdg.LocaleMatch(tmpExpandedAcronym.Lang) || tmpExpandedAcronym.Lang == "" && mimeType.ExpandedAcronym == "" {
					mimeType.ExpandedAcronym = tmpExpandedAcronym.Text
				}
			}

			if tmp.Icon.Name != "" {
				mimeType.IconUrl = link.IconUrl(tmp.Icon.Name)
			} else {
				mimeType.IconUrl = link.IconUrl(strings.Replace(mimeType.Id, "/", "-", -1))
			}

			for _, aliasStruct := range tmp.Alias {
				mimeType.Aliases = slice.AppendIfNotThere(mimeType.Aliases, aliasStruct.Type)
			}

			for _, tmpGlob := range tmp.Glob {
				mimeType.Globs = slice.AppendIfNotThere(mimeType.Globs, tmpGlob.Pattern)
			}

			for _, tmpSubClassOf := range tmp.SubClassOf {
				mimeType.SubClassOf = slice.AppendIfNotThere(mimeType.SubClassOf, tmpSubClassOf.Type)
			}

			if tmp.GenericIcon.Name != "" {
				mimeType.GenericIcon = tmp.GenericIcon.Name
			} else {
				slashPos := strings.Index(mimeType.Id, "/")
				mimeType.GenericIcon = mimeType.Id[:slashPos] + "-x-generic"
			}

			res[mimeType.Id] = mimeType
		}
	}

	// Do a transitive closure on 'SubClassOf'
	for _, mt := range res {
		for i := 0; i < len(mt.SubClassOf); i++ {
			if ancestor, ok := res[mt.SubClassOf[i]]; ok {
				for _, id := range ancestor.SubClassOf {
					mt.SubClassOf = slice.AppendIfNotThere(mt.SubClassOf, id)
				}
			}
		}
	}

	return res
}

func collectApplications(appdir string, apps map[string]*DesktopApplication) {
	var visitor = func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".desktop") {
			return nil
		}

		var id = strings.Replace(path[len(appdir) + 1:], "/", "-", -1)
		app, err := readDesktopFile(path, id)
		if err != nil {
			log.Warn("Error processing ", path, ":\n\t", err)
			return nil
		}

		if app.Hidden ||
			(len(app.OnlyShowIn) > 0 && !slice.ElementsInCommon(xdg.CurrentDesktop, app.OnlyShowIn)) ||
			(len(app.NotShowIn) > 0 && slice.ElementsInCommon(xdg.CurrentDesktop, app.NotShowIn)) {
			return nil
		}

		var executableName = app.Exec
		if lastSlash := strings.LastIndex(executableName, "/"); lastSlash > -1 {
			executableName = executableName[lastSlash:]
		}
		app.Keywords = append(app.Keywords, executableName)

		apps[app.Id] = app

		return nil
	}

	if xdg.DirOrFileExists(appdir) {
		_ = filepath.Walk(appdir, visitor)
	}
}

func readMimeappsList(path string, apps map[string]*DesktopApplication, defaultApps map[string][]string) {
	if iniFile, err := xdg.ReadIniFile(path); err == nil {
		if addedAssociations := iniFile.FindGroup("Added Associations"); addedAssociations != nil {
			for mimetypeId, appIds := range addedAssociations.Entries {
				for _, appId := range slice.Split(appIds, ";") {
					if app, ok := apps[appId]; ok {
						app.Mimetypes = slice.AppendIfNotThere(app.Mimetypes, mimetypeId)
					}
				}
			}
		}

		if removedAssociations := iniFile.FindGroup("Removed Associations"); removedAssociations != nil {
			for mimetypeId, appIds := range removedAssociations.Entries {
				for _, appId := range slice.Split(appIds, ";") {
					if app, ok := apps[appId]; ok {
						app.Mimetypes = slice.Remove(app.Mimetypes, mimetypeId)
					}
				}
				if defaultAppIds, ok := defaultApps[mimetypeId]; ok {
					for _, appId := range slice.Split(appIds, ";") {
						defaultAppIds = slice.Remove(defaultAppIds, appId)
					}
					defaultApps[mimetypeId] = defaultAppIds
				}
			}
		}

		if defaultApplications := iniFile.FindGroup("Default Applications"); defaultApplications != nil {
			for mimetypeId, defaultAppIds := range defaultApplications.Entries {
				var oldDefaultAppIds = defaultApps[mimetypeId]
				var newDefaultAppIds = make([]string, 0, len(defaultAppIds)+len(oldDefaultAppIds))
				for _, appId := range slice.Split(defaultAppIds, ";") {
					newDefaultAppIds = slice.AppendIfNotThere(newDefaultAppIds, appId)
				}
				for _, appId := range oldDefaultAppIds {
					newDefaultAppIds = slice.AppendIfNotThere(newDefaultAppIds, appId)
				}
				defaultApps[mimetypeId] = newDefaultAppIds
			}
		}
	}

}

func readDesktopFile(path string, id string) (*DesktopApplication, error) {
	if iniFile, err := xdg.ReadIniFile(path); err != nil {
		return nil, err
	} else if len(iniFile) == 0 || iniFile[0].Name != "Desktop Entry" {
		return nil, errors.New("file must start with '[Desktop Entry]'")
	} else {
		var da = DesktopApplication{ DesktopId: id}
		da.Id = id 
		da.Profile = "application"
		da.DesktopActions = []DesktopAction{}
		var actionNames = []string{}
		group := iniFile[0]

		if da.Type = group.Entries["Type"]; da.Type == "" {
			return nil, errors.New("desktop file invalid, no 'Type' given")
		}
		da.Version = group.Entries["Version"]
		if da.Title = group.Entries["Name"]; da.Title == "" {
			return nil, errors.New("desktop file invalid, no 'Name' given")
		}

		da.GenericName = group.Entries["GenericName"]
		da.NoDisplay = group.Entries["NoDisplay"] == "true"
		da.Comment = group.Entries["Comment"]
		da.IconUrl = link.IconUrl(group.Entries["Icon"])

		da.Hidden = group.Entries["Hidden"] == "true"
		da.OnlyShowIn = slice.Split(group.Entries["OnlyShowIn"], ";")
		da.NotShowIn = slice.Split(group.Entries["NotShowIn"], ";")
		da.DbusActivatable = group.Entries["DBusActivatable"] == "true"
		da.TryExec = group.Entries["TryExec"]
		da.Exec = group.Entries["Exec"]
		da.WorkingDir = group.Entries["Path"]
		da.Terminal = group.Entries["Terminal"] == "true"
		actionNames = slice.Split(group.Entries["Actions"], ";")
		da.Categories = slice.Split(group.Entries["Categories"], ";")
		da.Implements = slice.Split(group.Entries["Implements"], ";")
		da.Keywords = slice.Split(group.Entries["Keywords"], ";")
		da.StartupNotify = group.Entries["StartupNotify"] == "true"
		da.StartupWmClass = group.Entries["StartupWMClass"]
		da.Url = group.Entries["URL"]
		da.Mimetypes = slice.Split(group.Entries["MimeType"], ";")
		da.DesktopFile = path

		for _, actionGroup := range iniFile[1:] {
			if !strings.HasPrefix(actionGroup.Name, "Desktop Action ") {
				log.Warn(path, ", ", "Unknown group type: ", actionGroup.Name, " - ignoring\n")
			} else if currentAction := actionGroup.Name[15:]; !slice.Contains(actionNames, currentAction) {
				log.Warn(path, ", undeclared action: ", currentAction, " - ignoring\n")
			} else {
				var name = actionGroup.Entries["Name"]
				if name == "" {
					return nil, errors.New("Desktop file invalid, action " + actionGroup.Name + " has no default 'Name'")
				}

				da.DesktopActions = append(da.DesktopActions, DesktopAction{
					id:   currentAction,
					Name: name,
					Exec: actionGroup.Entries["Exec"],
					IconUrl: link.IconUrl(actionGroup.Entries["icon"]),
				})
			}
		}

		return &da, nil
	}

}
