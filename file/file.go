// Copyright (c) Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
package file

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rakyll/magicmime"
	"github.com/surlykke/RefudeServices/applications"
	"github.com/surlykke/RefudeServices/lib/link"
	"github.com/surlykke/RefudeServices/lib/log"
	"github.com/surlykke/RefudeServices/lib/relation"
	"github.com/surlykke/RefudeServices/lib/requests"
	"github.com/surlykke/RefudeServices/lib/resource"
	"github.com/surlykke/RefudeServices/lib/respond"
	"github.com/surlykke/RefudeServices/lib/searchutils"
	"github.com/surlykke/RefudeServices/lib/xdg"
)

func init() {
	if err := magicmime.Open(magicmime.MAGIC_MIME_TYPE | magicmime.MAGIC_SYMLINK | magicmime.MAGIC_ERROR); err != nil {
		panic(err)
	}
}

func getFileType(m os.FileMode) string {
	if m&fs.ModeDir > 0 {
		return "Directory"
	} else if m&fs.ModeSymlink > 0 {
		return "Symbolic link"
	} else if m&fs.ModeNamedPipe > 0 {
		return "Named pipe"
	} else if m&fs.ModeSocket > 0 {
		return "Socket"
	} else if m&fs.ModeDevice > 0 {
		return "Device"
	} else if m&fs.ModeCharDevice > 0 {
		return "Char device"
	} else if m&fs.ModeIrregular > 0 {
		return "Irregular"
	} else {
		return "File"
	}
}

type File struct {
	resource.ResourceData
	Name        string
	Type        string
	Permissions string
	Mimetype    string
	Apps        []string
}

func makeFileFromPath(path string) (*File, error) {
	var osPath = filepath.Clean("/" + path)
	if fileInfo, err := os.Stat(osPath); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		return makeFileFromInfo(osPath, fileInfo), nil
	}
}

func makeFileFromInfo(osPath string, fileInfo os.FileInfo) *File {
	var fileType = getFileType(fileInfo.Mode())
	var comment = osPath
	var mimetype, _ = magicmime.TypeByFile(osPath)
	var iconUrl = link.IconUrlFromName(strings.ReplaceAll(mimetype, "/", "-"))
	var f = File{
		ResourceData: *resource.MakeBase("/file/"+osPath[1:], fileInfo.Name(), comment, iconUrl, "file"),
		Type:         fileType,
		Permissions:  fileInfo.Mode().String(),
		Mimetype:     mimetype,
		Apps:         nil, // FIXMEapplications.GetAppsIds(mimetype),
	}


	if fileType == "Directory" {
		f.AddLink("/search?from="+f.Path, "", "", relation.Search)
	}

	fmt.Print("makefromInfo looking for '", mimetype, "'\n")

	var appDatas, ok = mimetypeAppDataMap[mimetype]
	fmt.Println("Got:", appDatas)
	if ok {
		for _, appData := range appDatas {
			f.Apps = append(f.Apps, appData.DesktopId)
			f.AddLink("?action="+appData.DesktopId, "Open with "+appData.Title, appData.IconUrl, relation.Action)
		}
	}

	return &f
}

func (f *File) Search(term string) []resource.Resource {
	var terms = strings.Split(term, "/")
	if f.Type == "Directory" {
		var rrList = make(resource.RRList, 0, 30)
		var osPath = f.Path[len("/file"):]
		search(&rrList, osPath, terms...)
		return rrList.GetResourcesSorted()
	} else {
		return []resource.Resource{}
	}
}


func searchFrom(dir, term string) resource.RRList {
	var collector = make(resource.RRList, 0, 100)
	var terms = strings.Split(term, "/")
	search(&collector, dir, terms...)
	return collector
}


func searchDesktop(term string) resource.RRList {
	var collector = make(resource.RRList, 0, 100)
	var terms = strings.Split(term, "/")
	if len(terms[0]) >= 3 {
		if rnk := searchutils.Match(term, xdg.Home); rnk >= 0 {
			if info, err := os.Stat(xdg.Home); err != nil {
				log.Warn(err)
			} else {
				var file = makeFileFromInfo(xdg.Home, info)
				collector = append(collector, resource.RankedResource{Res: file, Rank: rnk})
			}
		}
		search(&collector, xdg.Home, terms...)
		search(&collector, xdg.ConfigHome, terms...)
		search(&collector, xdg.DownloadDir, terms...)
		search(&collector, xdg.DocumentsDir, terms...)
		search(&collector, xdg.MusicDir, terms...)
		search(&collector, xdg.VideosDir, terms...)
	}
	return collector
}




// Assumes that dir is a directory an that len(terms) > 0
func search(collector *resource.RRList, dir string, terms ...string) {
	if file, err := os.Open(dir); err != nil {
		log.Warn(err)
	} else if entries, err := file.ReadDir(-1); err != nil {
		log.Warn(err)
	} else {
		for _, entry := range entries {
			if rnk := searchutils.Match(terms[0], entry.Name()); rnk > -1 {
				if len(terms) > 1 {
					if entry.IsDir() {
						search(collector, dir+"/"+entry.Name(), terms[1:]...)
					}
				} else {
					fileInfo, _ := entry.Info()
					var file = makeFileFromInfo(dir+"/"+entry.Name(), fileInfo)
					*collector = append(*collector, resource.RankedResource{Res: file, Rank: rnk})
				}
			}
		}
	}
}

func (f *File) DoPost(w http.ResponseWriter, r *http.Request) {
	var defaultAppId = ""
	if len(f.Apps) > 0 {
		defaultAppId = f.Apps[0]
	}
	var appId = requests.GetSingleQueryParameter(r, "action", defaultAppId)
	applications.OpenFile(appId, f.Path[len("/file"):])
	
	respond.Accepted(w)
}
