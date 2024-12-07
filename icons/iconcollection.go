package icons

import (
	"os"
	gopath "path"
	"path/filepath"
	"regexp"
	"slices"
	"sync"

	"github.com/surlykke/RefudeServices/lib/icon"
	"github.com/surlykke/RefudeServices/lib/log"
	"github.com/surlykke/RefudeServices/lib/path"
	"github.com/surlykke/RefudeServices/lib/repo"
	"github.com/surlykke/RefudeServices/lib/xdg"
)

type IconPath struct {
	Path    string
	MinSize uint32
	MaxSize uint32
}

var sessionIconsDir = xdg.RuntimeDir + "/org.refude.session-icons"

func init() {
	if err := os.MkdirAll(sessionIconsDir, 0700); err != nil {
		log.Warn("Unable to create ", sessionIconsDir, err)
	}
}

var themedIcons = make(map[icon.Name][]IconPath)
var sessionIcons = make(map[icon.Name][]IconPath)
var iconLock sync.Mutex

func getIconPaths(icon icon.Name) ([]IconPath, bool) {
	iconLock.Lock()
	defer iconLock.Unlock()
	if iconPaths, ok := themedIcons[icon]; ok {
		return iconPaths, true
	} else if iconPaths, ok = sessionIcons[icon]; ok {
		return iconPaths, true
	} else {
		return nil, false
	}

}

func addSessionIcon(icon icon.Name, iconPaths []IconPath) {
	iconLock.Lock()
	defer iconLock.Unlock()
	if _, ok := themedIcons[icon]; !ok {
		sessionIcons[icon] = iconPaths
	}
}

func addSessionIconSinglePath(icon icon.Name, path string) {
	addSessionIcon(icon, []IconPath{{Path: path, MinSize: 1, MaxSize: 1}})
}

func collectIcons() {
	var collected = make(map[icon.Name][]IconPath, 500)
	collectThemeIcons(collected)
	collectOtherIcons(collected)
	iconLock.Lock()
	defer iconLock.Unlock()
	themedIcons = collected
}

/*
Somewhat based on https://specifications.freedesktop.org/icon-theme-spec/icon-theme-spec-latest.html#icon_lookup .
icon scale is ignored (TODO)

We prefer an icon from theme with not-matching size over icon from parent theme with matching size. This should
give a gui a more consistent look
*/
func collectThemeIcons(collected map[icon.Name][]IconPath) {
	var searchOrder = determineSearchOrder()
	for _, themeId := range searchOrder {
		for name, iconPaths := range collectIconsFromTheme(themeId) {
			if _, ok := collected[name]; !ok {
				collected[name] = iconPaths
			}
		}
	}
}

func collectIconsFromTheme(themeId string) map[icon.Name][]IconPath {
	var iconsFromTheme = make(map[icon.Name][]IconPath)
	var theme, _ = repo.Get[*IconTheme](path.Of("/icontheme/", themeId))
	for _, basedir := range xdg.IconBasedirs {
		for _, themeDir := range theme.Dirs {
			var glob = basedir + "/" + themeId + "/" + themeDir.Path + "/*"
			if filePathsInThemeDir, err := filepath.Glob(glob); err == nil {
				for _, filePath := range filePathsInThemeDir {
					var ext = gopath.Ext(filePath)
					if ext == ".png" || ext == ".svg" { // TODO xpm
						var fileName = gopath.Base(filePath)
						var iconPath = IconPath{Path: filePath, MinSize: themeDir.MinSize, MaxSize: themeDir.MaxSize}
						var icon = icon.Name(fileName[0 : len(fileName)-4])
						iconsFromTheme[icon] = append(iconsFromTheme[icon], iconPath)
					}
				}
			}
		}
	}
	return iconsFromTheme
}

func collectOtherIcons(collected map[icon.Name][]IconPath) {
	var dirsToLookAt = make([]string, 0, len(xdg.IconBasedirs)+1)
	dirsToLookAt = append(dirsToLookAt, xdg.IconBasedirs...)
	dirsToLookAt = append(dirsToLookAt, xdg.PixmapDir)

	for _, dir := range dirsToLookAt {
		if filePathsInDir, err := filepath.Glob(dir + "/*"); err == nil {
			for _, filePath := range filePathsInDir {
				var ext = gopath.Ext(filePath)
				if ext == ".png" || ext == ".svg" { // TODO xpm
					var name = gopath.Base(filePath)
					var icon = icon.Name(name[0 : len(name)-4])
					if _, ok := collected[icon]; !ok { // We won't let a non-themed icon shadow a themed icon. cf. above.
						collected[icon] = []IconPath{{Path: filePath, MinSize: 1, MaxSize: 1}}
					}
				}

			}
		}
	}
}

func determineSearchOrder() []string {
	var searchOrder = make([]string, 0, 4)
	var walker func(themeId string)
	walker = func(themeId string) {
		if themeId != "" && themeId != "hicolor" && !slices.Contains(searchOrder, themeId) {
			if theme, ok := repo.Get[*IconTheme](path.Of("/icontheme/" + themeId)); ok {
				searchOrder = append(searchOrder, themeId)
				for _, parent := range theme.Inherits {
					walker(parent)
				}
			}
		}
	}
	walker(determineDefaultThemeId())
	searchOrder = append(searchOrder, "hicolor") // hicolor is the general fall back
	return searchOrder
}

func determineDefaultThemeId() string {
	var iconThemeDefPattern = regexp.MustCompile("gtk-icon-theme-name=(\\S+)")

	if defaultThemeId, ok := os.LookupEnv("REFUDE_ICON_THEME"); ok {
		return defaultThemeId
	} else {
		for _, iniFile := range []string{
			xdg.ConfigHome + "/gtk-4.0/settings.ini",
			"/etc/gtk-4.0/settings.ini",
			xdg.ConfigHome + "/gtk-3.0/settings.ini",
			"/etc/gtk-3.0/settings.ini",
			xdg.ConfigHome + "/gtk-2.0/settings.ini",
			"/etc/gtk-2.0/settings.ini",
			xdg.Home + "/.gtkrc-2.0", "/etc/gtk-2.0/gtkrc"} {

			if bytes, err := os.ReadFile(iniFile); err == nil {
				if matches := iconThemeDefPattern.FindStringSubmatch(string(bytes)); matches != nil {
					return matches[1]
				}
			}
		}
	}

	return ""
}
