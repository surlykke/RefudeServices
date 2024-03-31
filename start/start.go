// Copyright (c) Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
package start

import (
	"strings"

	"github.com/surlykke/RefudeServices/applications"
	"github.com/surlykke/RefudeServices/browsertabs"
	"github.com/surlykke/RefudeServices/file"
	"github.com/surlykke/RefudeServices/lib/link"
	"github.com/surlykke/RefudeServices/lib/resource"
	"github.com/surlykke/RefudeServices/lib/xdg"
	"github.com/surlykke/RefudeServices/notifications"
	"github.com/surlykke/RefudeServices/power"
	"github.com/surlykke/RefudeServices/watch"
	"github.com/surlykke/RefudeServices/wayland"
	"github.com/surlykke/RefudeServices/x11"
)

func onResourceChange() {
	watch.ResourceChanged("/start")
}

var Run = func() {
	notifications.Notifications.AddListener(onResourceChange)
	x11.Windows.AddListener(onResourceChange)
	wayland.Windows.AddListener(onResourceChange)
	applications.Applications.AddListener(onResourceChange)
	power.Devices.AddListener(onResourceChange)
}

type StartResource struct {
	resource.BaseResource
	searchTerm string
}

func (s *StartResource) Links(term string) link.List {
	return DoDesktopSearch(term)
}

func DoDesktopSearch(term string) link.List {
	var links = make(link.List, 0, 300)
	term = strings.ToLower(term)

	// Could perhaps be done concurrently..
	links = append(links, notifications.Notifications.Search(term, 0)...)
	if xdg.SessionType == "x11" {
		links = append(links, x11.Windows.Search(term, 0)...)
	} else {
		links = append(links, wayland.Windows.Search(term, 0)...)
	}
	links = append(links, applications.Applications.Search(term, 1)...)
	links = append(links, file.FileRepo.Search(term, 2)...)
	links = append(links, power.Devices.Search(term, 3)...)
	links = append(links, browsertabs.Tabs.Search(term, 0)...)

	return links
}

var Start = &StartResource{
	BaseResource: resource.BaseResource{Path: "/start", Title: "Start", Profile: "start"},
}

type BookmarksResource struct {
	resource.BaseResource
}

func (bm BookmarksResource) Links(searchTerm string) link.List {
	return link.List{
		{Href: "/application/", Title: "Applications", Profile: "application*"},
		{Href: "/window/", Title: "Windows", Profile: "window*"},
		{Href: "/notification/", Title: "Notifications", Profile: "notification*"},
		{Href: "/device/", Title: "Devices", Profile: "device*"},
		{Href: "/item/", Title: "Items", Profile: "item*"}}
}

var Bookmarks = &BookmarksResource{BaseResource: resource.BaseResource{Path: "/bookmarks", Title: "Bookmarks", Profile: "bookmarks"}}



