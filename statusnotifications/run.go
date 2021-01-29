// Copyright (c) 2017,2018,2019 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package statusnotifications

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"sync"

	"github.com/surlykke/RefudeServices/watch"

	"github.com/godbus/dbus/v5"
	"github.com/surlykke/RefudeServices/lib/respond"
)

var itemPathPattern = regexp.MustCompile("^(/item/[^/]+)(/menu)?")

func Handler(r *http.Request) http.Handler {
	if r.URL.Path == "/items" {
		return Collect()
	} else if match := itemPathPattern.FindStringSubmatch(r.URL.Path); match != nil {
		if item := get(match[1]); item == nil {
			return nil
		} else if match[2] == "/menu" {
			if item.menuPath == "" {
				return nil
			} else {
				return item.buildMenu()
			}
		} else {
			return item
		}
	} else {
		return nil
	}
}

func Collect() respond.Links {
	lock.Lock()
	defer lock.Unlock()
	var links = make(respond.Links, 0, len(items))
	for _, item := range items {
		links = append(links, item.Link())
	}
	sort.Sort(links)
	return links
}

func AllPaths() []string {
	lock.Lock()
	defer lock.Unlock()
	var paths = make([]string, 0, 2*len(items)+1)
	for path, item := range items {
		paths = append(paths, path)
		if item.Menu != "" {
			paths = append(paths, item.Menu)
		}
	}
	paths = append(paths, "/items")
	return paths
}

func Run() {
	getOnTheBus()
	go monitorSignals()

	// TODO After a restart, pick up those that where?

	for event := range events {
		switch event.eventName {
		case "ItemCreated":
			set(buildItem(event.sender, event.path))
		case "ItemRemoved":
			remove(event.sender, event.path)
		default:
			var path = itemSelf(event.sender, event.path)
			if item := get(path); item != nil {
				var tmp = *item
				var itemCopy = &tmp
				switch event.eventName {
				case "org.kde.StatusNotifierItem.NewTitle":
					fmt.Println("NewTitle")
					updateTitle(itemCopy)
				case "org.kde.StatusNotifierItem.NewStatus":
					fmt.Println("NewStatus")
					updateStatus(itemCopy)
				case "org.kde.StatusNotifierItem.NewToolTip":
					fmt.Println("NewToolTip")
					updateToolTip(itemCopy)
				case "org.kde.StatusNotifierItem.NewIcon":
					fmt.Println("NewIcon")
					updateIcon(itemCopy)
				case "org.kde.StatusNotifierItem.NewIconThemePath":
					fmt.Println("NewIconThemePath")
					updateIconThemePath(itemCopy)
				case "org.kde.StatusNotifierItem.NewAttentionIcon":
					fmt.Println("NewAttentionIcon")
					updateAttentionIcon(itemCopy)
				case "org.kde.StatusNotifierItem.NewOverlayIcon":
					fmt.Println("NewOverlayIcon")
					updateOverlayIcon(itemCopy)
				default:
					continue
				}
				set(itemCopy)
			} else {
				continue
			}
		}
	}
}

var items = make(ItemMap)
var lock sync.Mutex

func set(item *Item) {
	var self = itemSelf(item.sender, item.itemPath)
	lock.Lock()
	items[self] = item
	lock.Unlock()
	sendEvent(self)
	sendEvent("/items")
}

func get(path string) *Item {
	lock.Lock()
	defer lock.Unlock()
	return items[path]
}

func remove(sender string, itemPath dbus.ObjectPath) {
	var self = itemSelf(sender, itemPath)
	lock.Lock()
	delete(items, self)
	lock.Unlock()
	sendEvent(self)
	sendEvent("/items")
}

func sendEvent(path string) {
	watch.SomethingChanged(path)
}
