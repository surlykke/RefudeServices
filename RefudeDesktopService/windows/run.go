// Copyright (c) 2017,2018,2019 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package windows

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync/atomic"

	"github.com/surlykke/RefudeServices/RefudeDesktopService/windows/xlib"
	"github.com/surlykke/RefudeServices/lib/respond"
)

const (
	NET_WM_VISIBLE_NAME      = "_NET_WM_VISIBLE_NAME"
	NET_WM_NAME              = "_NET_WM_NAME"
	WM_NAME                  = "WM_NAME"
	NET_WM_ICON              = "_NET_WM_ICON"
	NET_CLIENT_LIST_STACKING = "_NET_CLIENT_LIST_STACKING"
	NET_WM_STATE             = "_NET_WM_STATE"
)

var windowPath = regexp.MustCompile("^/window/(\\d+)(/screenshot)?$")

func Handler(r *http.Request) http.Handler {
	if r.URL.Path == "/windows" {
		return Windows()
	} else if matches := windowPath.FindStringSubmatch(r.URL.Path); matches == nil {
		return nil
	} else if val, err := strconv.ParseUint(matches[1], 10, 32); err != nil {
		return nil
	} else {
		var id = uint32(val)
		var screenShot = matches[2] != ""
		for _, wId := range windows.Load().([]uint32) {
			if id == wId {
				if screenShot {
					return ScreenShot(id)
				} else {
					return Window(id)
				}
			}
		}

	}
	return nil
}

func Windows() respond.StandardFormatList {
	var idList = windows.Load().([]uint32)
	var sfl = make(respond.StandardFormatList, 0, len(idList))
	for i, id := range idList {
		var sf = Window(id).ToStandardFormat()
		sf.Rank = 10 * uint(i)
		sfl = append(sfl, sf)
	}
	return sfl
}

func AllPaths() []string {
	var windowList = windows.Load().([]uint32)
	var paths = make([]string, 0, 2*len(windowList)+1)
	for _, window := range windowList {
		paths = append(paths, fmt.Sprintf("/window/%d", window))
		paths = append(paths, fmt.Sprintf("/window/%d/screenshot", window))
	}
	paths = append(paths, "/windows")
	return paths
}

// Maintains windows lists
func Run() {
	var eventConnection = xlib.MakeConnection()
	eventConnection.SubscribeToStackEvents()

	for {
		if wIds, err := eventConnection.GetUint32s(0, NET_CLIENT_LIST_STACKING); err != nil {
			log.Println("WARN: Unable to retrieve _NET_CLIENT_LIST_STACKING", err)
			windows.Store([]uint32{})
		} else {
			var list = make([]uint32, len(wIds), len(wIds))
			// Revert so highest in stack comes first
			for i := 0; i < len(wIds); i++ {
				list[i] = wIds[len(wIds)-i-1]
			}
			windows.Store(list)
		}

		eventConnection.WaitforStackEvent()
	}
}

var windows atomic.Value

func init() {
	windows.Store([]uint32{})
}
