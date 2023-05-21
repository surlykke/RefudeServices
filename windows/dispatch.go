package windows

import (

	"github.com/surlykke/RefudeServices/lib/resource"
	"github.com/surlykke/RefudeServices/lib/xdg"
	"github.com/surlykke/RefudeServices/wayland"
	"github.com/surlykke/RefudeServices/x11"
)

var runningX11 = xdg.SessionType == "x11"

func GetWindowCollection() resource.ResourceRepo {
	if runningX11 {
		return x11.Windows
	} else {
		return wayland.Windows 
	}
}

func Run() {
	if runningX11 {
		x11.Run()
	} else {
		wayland.Run()
	}
}

func GetPaths() []string {
	if runningX11 {
		return x11.Windows.GetPaths()
	} else {
		return 	wayland.Windows.GetPaths()
	}
}


func PurgeAndShow(applicationTitle string, focus bool) bool {
	if runningX11 {
		return x11.PurgeAndShow(applicationTitle, focus) 
	} else {
		return wayland.PurgeAndShow(applicationTitle, focus)
	}
}

func PurgeAndHide(applicationTitle string) {
	if runningX11 {
		x11.PurgeAndHide(applicationTitle)
	} else {
		wayland.PurgeAndHide(applicationTitle)
	}
}

func MoveAndResize(applicationTitle string, x,y int32, width,height uint32, focus bool) bool {
	if runningX11 {
		return x11.MoveAndResize(applicationTitle, x,y, width, height, focus) 
	} else {
		return wayland.MoveAndResize(applicationTitle, x, y, width, height) 
	}
}


