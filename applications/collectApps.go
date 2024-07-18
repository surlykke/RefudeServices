package applications

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/surlykke/RefudeServices/lib/link"
	"github.com/surlykke/RefudeServices/lib/log"
	"github.com/surlykke/RefudeServices/lib/relation"
	"github.com/surlykke/RefudeServices/lib/resource"
	"github.com/surlykke/RefudeServices/lib/slice"
	"github.com/surlykke/RefudeServices/lib/xdg"
)

func collectApps() (map[string]*DesktopApplication, map[string][]string) {
	// ----
	// Keyed by apps DeskoptId, ie. 'firefox.desktop' not '/application/firefox.desktop'
	var apps = make(map[string]*DesktopApplication)
	// Mimetype id (ie. 'text/html' not '/mimetype/text/html') to list of DesktopIds
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
	return apps, defaultApps
}

func collectApplications(applicationsDir string, apps map[string]*DesktopApplication) {
	var visitor = func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(filePath, ".desktop") {
			return nil
		}

		var id = strings.Replace(filePath[len(applicationsDir)+1:], "/", "-", -1)
		app, err := readDesktopFile(filePath, stripDesktopSuffix(id))
		if err != nil {
			log.Warn("Error processing ", filePath, ":\n\t", err)
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

		apps[app.DesktopId] = app

		return nil
	}

	if xdg.DirOrFileExists(applicationsDir) {
		_ = filepath.Walk(applicationsDir, visitor)
	}
}

func readMimeappsList(path string, apps map[string]*DesktopApplication, defaultApps map[string][]string) {
	if iniFile, err := xdg.ReadIniFile(path); err == nil {
		if addedAssociations := iniFile.FindGroup("Added Associations"); addedAssociations != nil {
			for mimetypeId, appIds := range addedAssociations.Entries {
				for _, appId := range slice.Split(appIds, ";") {
					appId = stripDesktopSuffix(appId)
					if app, ok := apps[appId]; ok {
						app.Mimetypes = slice.AppendIfNotThere(app.Mimetypes, mimetypeId)
					}
				}
			}
		}

		if removedAssociations := iniFile.FindGroup("Removed Associations"); removedAssociations != nil {
			for mimetypeId, appIds := range removedAssociations.Entries {
				for _, appId := range slice.Split(appIds, ";") {
					appId = stripDesktopSuffix(appId)
					if app, ok := apps[appId]; ok {
						app.Mimetypes = slice.Remove(app.Mimetypes, mimetypeId)
					}
				}
				if defaultAppIds, ok := defaultApps[mimetypeId]; ok {
					for _, appId := range slice.Split(appIds, ";") {
						appId = stripDesktopSuffix(appId)
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
					appId = stripDesktopSuffix(appId)
					newDefaultAppIds = slice.AppendIfNotThere(newDefaultAppIds, appId)
				}
				for _, appId := range oldDefaultAppIds {
					appId = stripDesktopSuffix(appId)
					newDefaultAppIds = slice.AppendIfNotThere(newDefaultAppIds, appId)
				}
				defaultApps[mimetypeId] = newDefaultAppIds
			}
		}
	}

}

func readDesktopFile(filePath string, id string) (*DesktopApplication, error) {
	if iniFile, err := xdg.ReadIniFile(filePath); err != nil {
		return nil, err
	} else if len(iniFile) == 0 || iniFile[0].Name != "Desktop Entry" {
		return nil, errors.New("file must start with '[Desktop Entry]'")
	} else {
		group := iniFile[0]
		var path, title, comment = "/application/" + id, group.Entries["Name"], group.Entries["Comment"]
		var da = DesktopApplication{
			ResourceData: *resource.MakeBase(path, title, comment, "", "application"),
			DesktopId:    id,
		}

		if da.Title == "" {
			return nil, errors.New("desktop file invalid, no 'Name' given")
		}

		if da.Type = group.Entries["Type"]; da.Type == "" {
			return nil, errors.New("desktop file invalid, no 'Type' given")
		}
		da.Version = group.Entries["Version"]
		da.GenericName = group.Entries["GenericName"]
		da.IconUrl = link.IconUrlFromName(group.Entries["Icon"])
		da.NoDisplay = group.Entries["NoDisplay"] == "true"
		da.Hidden = group.Entries["Hidden"] == "true"
		da.OnlyShowIn = slice.Split(group.Entries["OnlyShowIn"], ";")
		da.NotShowIn = slice.Split(group.Entries["NotShowIn"], ";")
		da.DbusActivatable = group.Entries["DBusActivatable"] == "true"
		da.TryExec = group.Entries["TryExec"]
		da.Exec = group.Entries["Exec"]
		da.WorkingDir = group.Entries["Path"]
		da.Terminal = group.Entries["Terminal"] == "true"
		da.Categories = slice.Split(group.Entries["Categories"], ";")
		da.Implements = slice.Split(group.Entries["Implements"], ";")
		da.Keywords = slice.Split(group.Entries["Keywords"], ";")
		da.StartupNotify = group.Entries["StartupNotify"] == "true"
		da.StartupWmClass = group.Entries["StartupWMClass"]
		da.Url = group.Entries["URL"]
		da.Mimetypes = slice.Split(group.Entries["MimeType"], ";")
		da.DesktopFile = filePath

		da.Links = da.Links.Add(da.Path, "Launch", da.IconUrl, relation.Action, "")
		da.DesktopActions = []DesktopAction{}
		var actionNames = slice.Split(group.Entries["Actions"], ";")

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
				var iconUrl = link.IconUrlFromName(actionGroup.Entries["icon"])
				da.DesktopActions = append(da.DesktopActions, DesktopAction{
					id:      currentAction,
					Name:    name,
					Exec:    actionGroup.Entries["Exec"],
					IconUrl: iconUrl,
				})
				da.Links = da.Links.Add(da.Path+"?action="+currentAction, name, iconUrl, relation.Action, "")
			}
		}

		return &da, nil
	}

}

var desktopSuffix = regexp.MustCompile(`\.desktop$`)

func stripDesktopSuffix(fileName string) string {
	return desktopSuffix.ReplaceAllString(fileName, "")
}
