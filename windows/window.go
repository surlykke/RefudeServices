// Copyright (c) Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package windows

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/surlykke/RefudeServices/icons"
	"github.com/surlykke/RefudeServices/lib/link"
	"github.com/surlykke/RefudeServices/lib/log"
	"github.com/surlykke/RefudeServices/lib/relation"
	"github.com/surlykke/RefudeServices/lib/requests"
	"github.com/surlykke/RefudeServices/lib/resource"
	"github.com/surlykke/RefudeServices/lib/respond"
	"github.com/surlykke/RefudeServices/lib/searchutils"
	"github.com/surlykke/RefudeServices/windows/x11"
)

type Bounds struct {
	X, Y int32
	W, H uint32
}

type Window struct {
	Id       uint32
	Name     string
	IconName string `json:",omitempty"`
	State    x11.WindowStateMask
	Stacking int // 0 means: on top, then 1, then 2 etc. -1 means we don't know (yet)
}

func (w *Window) Self() string {
	return fmt.Sprintf("/window/%d", w.Id)
}

func (w *Window) Presentation() (title string, comment string, icon link.Href, profile string) {
	return w.Name, "", link.IconUrl(w.IconName), "window"
}

func (w *Window) Links(context string) link.List {
	if searchutils.Match(context, w.Name) > -1 {
		return link.List{
			link.Make(context+w.Self(), w.Name, w.IconName, relation.DefaultAction),
			link.Make(context+w.Self(), w.Name, w.IconName, relation.Delete),
		}
	} else {
		return link.List{} 
	}
}

// Caller ensures thread safety (calls to x11)
func makeWindow(p x11.Proxy, wId uint32) *Window {
	var win = &Window{Id: wId}
	win.Name, _ = x11.GetName(p, wId)
	win.IconName, _ = GetIconName(p, wId)
	win.State = x11.GetStates(p, wId)
	win.Stacking = -1
	return win
}

func (win *Window) DoDelete(w http.ResponseWriter, r *http.Request) {
	requestProxyMutex.Lock()
	x11.CloseWindow(requestProxy, win.Id)
	requestProxyMutex.Unlock()
	respond.Accepted(w)
}

func (win *Window) DoPost(w http.ResponseWriter, r *http.Request) {
	var action = requests.GetSingleQueryParameter(r, "action", "")
	if performAction(win.Id, action) {
		respond.Accepted(w)
	} else {
		respond.NotFound(w)
	}
}

func performAction(wId uint32, action string) bool {
	requestProxyMutex.Lock()
	defer requestProxyMutex.Unlock()

	var found = true
	if action == "" {
		x11.RaiseAndFocusWindow(requestProxy, wId)
	} else if action == "restore" {
		x11.RemoveStates(requestProxy, wId, x11.HIDDEN|x11.MAXIMIZED_HORZ|x11.MAXIMIZED_VERT)
	} else if action == "minimize" {
		x11.AddStates(requestProxy, wId, x11.HIDDEN)
	} else if action == "maximize" {
		x11.AddStates(requestProxy, wId, x11.MAXIMIZED_HORZ|x11.MAXIMIZED_VERT)
	} else if strings.HasPrefix(action, "move::") {
		found = false
		var monitorName = action[6:]
		for _, m := range getDesktopLayout().Monitors {
			if monitorName == m.Name {
				var marginW, marginH = m.W / 10, m.H / 10
				var saveStates = x11.GetStates(requestProxy, wId) & (x11.HIDDEN | x11.MAXIMIZED_HORZ | x11.MAXIMIZED_VERT)
				x11.RemoveStates(requestProxy, wId, x11.HIDDEN|x11.MAXIMIZED_HORZ|x11.MAXIMIZED_VERT)
				x11.SetBounds(requestProxy, wId, m.X+int32(marginW), m.Y+int32(marginH), m.W-2*marginW, m.H-2*marginH)
				x11.AddStates(requestProxy, wId, saveStates)
				found = true
				break
			}
		}
	}
	return found
}

func RaiseAndFocusNamedWindow(name string) bool {
	if res := Windows.FindFirst(func(res resource.Resource) bool { return res.(*Window).Name == name }); res != nil {
		x11.RaiseAndFocusWindow(requestProxy, res.(*Window).Id)
		return true
	} else {
		return false
	}
}

func ResizeNamedWindow(name string, newWidth, newHeight uint32) bool {
	if res := Windows.FindFirst(func(res resource.Resource) bool { return res.(*Window).Name == name }); res != nil {
		x11.Resize(requestProxy, res.(*Window).Id, newWidth, newHeight)
		return true
	} else {
		return false
	}
}

func relevantForDesktopSearch(w *Window) bool {
	return w.State&(x11.SKIP_TASKBAR|x11.SKIP_PAGER|x11.ABOVE) == 0
}

func GetIconName(p x11.Proxy, wId uint32) (string, error) {
	pixelArray, err := x11.GetIcon(p, wId)
	if err != nil {
		log.Warn("Error converting x11 icon to pngs", err)
		return "", err
	} else {
		return icons.AddX11Icon(pixelArray)
	}
}
