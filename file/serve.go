// Copyright (c) Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
package file

import (
	"strings"

	"github.com/surlykke/RefudeServices/applications"
	"github.com/surlykke/RefudeServices/lib/log"
	"github.com/surlykke/RefudeServices/lib/repo"
	"github.com/surlykke/RefudeServices/lib/resource"
)

var collections = applications.SubscribeToCollections()

var apps = make(map[string]*applications.DesktopApplication)
var mimetypes = make(map[string]*applications.Mimetype)

func Run() {
	var repoRequests = repo.MakeAndRegisterRequestChan()
	for {
		select {
		case req := <-repoRequests:
			switch req.ReqType {
			case repo.ByPath:
				if strings.HasPrefix(req.Data, "/file") {
					if f := GetResource(req.Data); f != nil {
						req.Replies <- resource.RankedResource{Res: f}
					}
				}
			case repo.Search:
				for _, rr := range searchDesktop(req.Data) {
					req.Replies <- rr
				}
			}
			req.Wg.Done()
		case collection := <-collections:
			apps = collection.Apps
			mimetypes = collection.Mimetypes
		}

	}
}

func GetResource(path string) *File {
	if !strings.HasPrefix(path, "/file/") {
		log.Warn("Unexpeded path:", path)
		return nil
	} else if file, err := makeFileFromPath(path[5:]); err != nil {
		log.Warn("Could not make file from", path[5:], err)
		return nil
	} else if file == nil {
		return nil
	} else {
		return file
	}
}
