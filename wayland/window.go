package wayland

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/surlykke/RefudeServices/lib/link"
	"github.com/surlykke/RefudeServices/lib/requests"
	"github.com/surlykke/RefudeServices/lib/resource"
	"github.com/surlykke/RefudeServices/lib/resourcerepo"
	"github.com/surlykke/RefudeServices/lib/respond"
)

// Get current rect
//swaymsg -t get_tree | jq '.nodes[1].nodes[].floating_nodes[] | select(.name="org.refude.panel") | (.rect)'

// focus
// swaymsg '[title=org.refude.panel] focus'

// Move to
// swaymsg '[title=org.refude.panel] move absolute position 1200 0'

//
// swaymsg "[title=org.refude.panel] resize set width 200" (or height)

type WindowStateMask uint8

const (
	MAXIMIZED = 1 << iota
	MINIMIZED
	ACTIVATED
	FULLSCREEN
)

func (wsm WindowStateMask) Is(other WindowStateMask) bool {
	return wsm&other == other
}

func (wsm WindowStateMask) MarshalJSON() ([]byte, error) {
	var list = make([]string, 0, 4)
	if wsm&MAXIMIZED > 0 {
		list = append(list, "MAXIMIZED")
	}
	if wsm&MINIMIZED > 0 {
		list = append(list, "MINIMIZED")
	}
	if wsm&ACTIVATED > 0 {
		list = append(list, "ACTIVATED")
	}
	if wsm&FULLSCREEN > 0 {
		list = append(list, "FULLSCREEN")
	}
	return json.Marshal(list)
}

type WaylandWindow struct {
	resource.BaseResource
	Wid   uint64 `json:"-"`
	AppId string `json:"app_id"`
	State WindowStateMask
}

func MakeWindow(wId uint64) *WaylandWindow {
	return &WaylandWindow{
		BaseResource: resource.BaseResource{
			Path:     fmt.Sprintf("/window/%d", wId),
			Profile: "window",
		},
		Wid: wId,
	}
}

func (this *WaylandWindow) RelevantForSearch(term string) bool {
	return !strings.HasPrefix(this.Title, "Refude launcher")
}

func (this *WaylandWindow) Actions() link.ActionList {
	return link.ActionList{
		{Name: "activate", Title: "Raise and focus", IconUrl: this.IconUrl},
	}
}

func (this *WaylandWindow) DoDelete(w http.ResponseWriter, r *http.Request) {
	close(this.Wid)
	respond.Accepted(w)
}

func (this *WaylandWindow) DoPost(w http.ResponseWriter, r *http.Request) {
	var action = requests.GetSingleQueryParameter(r, "action", "activate")
	if "activate" == action {
		activate(this.Wid)
		respond.Accepted(w)
	} else {
		respond.NotFound(w)
	}
}

var recentMap = make(map[uint64]uint32)
var recentCount uint32
var recentMapLock sync.Mutex

func getCopy(wId uint64) *WaylandWindow {
	var copy WaylandWindow
	var path = fmt.Sprintf("/window/%d", wId)
	if w, ok := resourcerepo.GetTyped[*WaylandWindow](path); ok {
		copy = *w
	} else {
		copy = *MakeWindow(wId)
	}
	return &copy
}

func Run() {
	//go watchApplications()
	setupAndRunAsWaylandClient()
}

/*func watchApplications() {
	applications.Applications.AddListener(func() {
		for _, w := range Windows.GetAll() {
			var win = *w
			win.IconUrl = applications.GetIconUrl(win.AppId + ".desktop")
			Windows.Update(&win)
		}
	})
}*/ // TODO

var rememberedActive uint64 = 0
var rememberedActiveLock sync.Mutex

func RememberActive() {
	if active := resourcerepo.FindTypedUnderPrefix[*WaylandWindow]("/window/", func(w *WaylandWindow) bool {return w.State.Is(ACTIVATED) }); len(active) > 0 {
		rememberedActiveLock.Lock()
		rememberedActive = active[0].Wid
		rememberedActiveLock.Unlock()
	}
}

func ActivateRememberedActive() {
	rememberedActiveLock.Lock()
	var copy = rememberedActive
	rememberedActiveLock.Unlock()
	if copy > 0 {
		activate(copy)
	}
}