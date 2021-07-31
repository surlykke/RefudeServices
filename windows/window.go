// Copyright (c) 2017 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package windows

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/surlykke/RefudeServices/icons"
	"github.com/surlykke/RefudeServices/lib/log"
	"github.com/surlykke/RefudeServices/lib/requests"
	"github.com/surlykke/RefudeServices/lib/respond"
	"github.com/surlykke/RefudeServices/windows/x11"
)

type Bounds struct {
	X, Y int32
	W, H uint32
}

type Window struct {
	respond.Resource
	Id       uint32
	Name     string
	IconName string `json:",omitempty"`
	State    x11.WindowStateMask
	Stacking int // 0 means: on top, then 1, then 2 etc. -1 means we don't know (yet)
}

// Caller ensures thread safety (calls to x11)
func BuildWindow(p x11.Proxy, wId uint32) *Window {
	var win = &Window{Id: wId}
	win.Name, _ = x11.GetName(p, wId)
	win.IconName, _ = GetIconName(p, wId)
	win.State = x11.GetStates(p, wId)
	win.Stacking = -1
	win.Resource = respond.MakeResource(fmt.Sprintf("/window/%d", wId), win.Name, icons.IconUrl(win.IconName), "window")
	return win
}

func (win *Window) DoDelete(w http.ResponseWriter, r *http.Request) {
	requestProxyMutex.Lock()
	x11.CloseWindow(requestProxy, win.Id)
	defer requestProxyMutex.Unlock()
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
	if action == "" {
		x11.RaiseAndFocusWindow(requestProxy, wId)
	} else if action == "restore" {
		x11.RemoveStates(requestProxy, wId, x11.HIDDEN|x11.MAXIMIZED_HORZ|x11.MAXIMIZED_VERT)
	} else if action == "minimize" {
		x11.AddStates(requestProxy, wId, x11.HIDDEN)
	} else if action == "maximize" {
		x11.AddStates(requestProxy, wId, x11.MAXIMIZED_HORZ|x11.MAXIMIZED_VERT)
	} else if strings.HasPrefix(action, "move::") {
		var monitorName = action[6:]
		for _, m := range repo.getDesktopLayout().Monitors {
			if monitorName == m.Name {
				var marginW, marginH = m.W / 10, m.H / 10
				var saveStates = x11.GetStates(requestProxy, wId) & (x11.HIDDEN | x11.MAXIMIZED_HORZ | x11.MAXIMIZED_VERT)
				x11.RemoveStates(requestProxy, wId, x11.HIDDEN|x11.MAXIMIZED_HORZ|x11.MAXIMIZED_VERT)
				x11.SetBounds(requestProxy, wId, m.X+int32(marginW), m.Y+int32(marginH), m.W-2*marginW, m.H-2*marginH)
				x11.AddStates(requestProxy, wId, saveStates)
				return true
			}
		}
		return false
	}
	return true
}

func updateLinks(win *Window, desktopLayout *DesktopLayout) {
	win.ClearNonSelfLinks()
	win.AddDefaultActionLink("Raise and focus", "")
	win.AddDeleteLink("Close", "")
	if win.State.Is(x11.HIDDEN) || win.State.Is(x11.MAXIMIZED_HORZ|x11.MAXIMIZED_VERT) {
		win.AddActionLink("Restore window", "", "restore")
	} else {
		win.AddActionLink("Minimize window", "", "minimize")
		win.AddActionLink("Maximize window", "", "maximize")
	}

	for _, m := range desktopLayout.Monitors {
		var actionId = url.QueryEscape("move::" + m.Name)
		win.AddActionLink("Move to monitor "+m.Name, "", actionId)
	}
}

func (win *Window) copy() *Window {
	var result = &Window{}
	*result = *win
	return result
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

var noBounds = &Bounds{0, 0, 0, 0}

func GetBounds(wId uint32) *Bounds {
	requestProxyMutex.Lock()
	defer requestProxyMutex.Unlock()
	// TODO Perhaps some caching
	if x, y, w, h, err := x11.GetGeometry(requestProxy, wId); err != nil {
		return noBounds
	} else {
		return &Bounds{x, y, w, h}
	}
}

type WindowStack []*Window

// Implement sort.Interface
func (ws WindowStack) Len() int {
	return len(ws)
}

func (ws WindowStack) Less(i, j int) bool {
	return ws[i].Stacking < ws[j].Stacking
}

func (ws WindowStack) Swap(i, j int) {
	ws[i], ws[j] = ws[j], ws[i]
}
