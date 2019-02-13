// Copyright (c) 2017 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package statusnotifications

import (
	"errors"
	"fmt"
	"github.com/godbus/dbus"
	"github.com/godbus/dbus/introspect"
	"github.com/godbus/dbus/prop"
	"github.com/surlykke/RefudeServices/lib/dbusutils"
	"github.com/surlykke/RefudeServices/lib/image"
	"github.com/surlykke/RefudeServices/lib/resource"
	"github.com/surlykke/RefudeServices/lib/server"
	"github.com/surlykke/RefudeServices/lib/slice"
	"log"
	"reflect"
	"regexp"
	"strings"
	"time"
)

var itemCollection = MakeItemCollection()
var ItemServer = server.MakeServer(itemCollection)

func Run() {
	getOnTheBus()
	go monitorSignals()

	for event := range events {
		fmt.Println("Got event: ", event)
		var key = event.sender + string(event.path)
		switch event.eventType {
		case ItemCreated:
			item := MakeItem(event.sender, event.path)
			item.AbstractResource = resource.MakeAbstractResource(resource.StandardizedPath("/item/" + item.key), ItemMediaType)
			updateItem(item)
			if item.menuPath != "" {
				fetchMenu(item)
			}
			itemCollection.Lock()
			itemCollection.items[key] = item
			updateWatcherProperties()
			itemCollection.JsonResponseCache.ClearByPrefixes("/item/" + key, "/items")
			itemCollection.Unlock()
			go monitorItem(event.sender, event.path)
		case ItemRemoved:
			itemCollection.Lock()
			delete(itemCollection.items, key)
			updateWatcherProperties()
			itemCollection.JsonResponseCache.ClearByPrefixes("/item/" + key, "/items")
			itemCollection.Unlock()
		case ItemUpdated:
			itemCollection.Lock()
			if item, ok := itemCollection.items[key]; ok {

				itemCollection.Unlock()
				var copy = *item
				updateItem(&copy)
				itemCollection.Lock()

				itemCollection.items[key] = &copy
				itemCollection.JsonResponseCache.ClearByPrefixes("/item/" + key, "/items")
			}
			itemCollection.Unlock()
		case MenuUpdated:
			itemCollection.Lock()
			var item = itemCollection.findByMenuPath(event.sender, event.path);
			if item != nil {
				itemCollection.Unlock()
				var copy = *item
				fetchMenu(&copy)
				itemCollection.Lock()

				itemCollection.items[copy.key] = &copy
				itemCollection.JsonResponseCache.ClearByPrefixes("/item/" + copy.key, "/items")
			}
			itemCollection.Unlock()
		}
	}
}





const WATCHER_SERVICE = "org.kde.StatusNotifierWatcher"
const WATCHER_PATH = "/StatusNotifierWatcher"
const WATCHER_INTERFACE = WATCHER_SERVICE
const HOST_SERVICE = "org.kde.StatusNotifierHost"
const ITEM_PATH = "/StatusNotifierItem"
const ITEM_INTERFACE = "org.kde.StatusNotifierItem"
const MENU_INTERFACE = "com.canonical.dbusmenu"

var conn *dbus.Conn
var watcherProperties *prop.Properties

func senderAndPath(serviceName string, sender dbus.Sender) (string, dbus.ObjectPath) {
	if regexp.MustCompile("^(/\\w+)+$").MatchString(serviceName) {
		return string(sender), dbus.ObjectPath(serviceName)
	} else {
		return string(sender), dbus.ObjectPath(ITEM_PATH)
	}
}

/**
 * serviceId Can be a name of service or a path of object
 */
func addItem(serviceName string, sender dbus.Sender) *dbus.Error {
	fmt.Println("addItem:", serviceName, ",", sender)
	var event = Event{eventType: ItemCreated}
	event.sender, event.path = senderAndPath(serviceName, sender)
	events <- event
	return nil
}

func monitorSignals() {
	var dbusSignals = make(chan *dbus.Signal, 50)
	conn.Signal(dbusSignals)
	addMatch := "org.freedesktop.DBus.AddMatch"
	conn.BusObject().Call(addMatch, 0, "type='signal', interface='org.kde.StatusNotifierItem'")
	conn.BusObject().Call(addMatch, 0, "type='signal', interface='com.canonical.dbusmenu'")

	for signal := range dbusSignals {
		if strings.HasPrefix(signal.Name, "org.kde.StatusNotifierItem.New") {
			events <- Event{eventType: ItemUpdated, sender: signal.Sender, path: signal.Path}
		} else if signal.Name == "com.canonical.dbusmenu.LayoutUpdated" {
			events <- Event{eventType: MenuUpdated, sender: signal.Sender, path: signal.Path}
		}
	}
}

// We cannot reliably detect items disappearing by signals, it seems, so we do brute force polling
func monitorItem(sender string, itemPath dbus.ObjectPath) {
	fmt.Println("itemWatcher: ", sender, itemPath)
	for {
		if _, ok := dbuscall.GetSingleProp(conn, sender, itemPath, ITEM_INTERFACE, "Status"); !ok {
			break
		}
		time.Sleep(time.Second)
	}
	events <- Event{eventType: ItemRemoved, sender: sender, path: itemPath}
}

// ----------------------------------------------------------------------------------------------------

func getOnTheBus() {
	var err error

	// Get on the bus
	conn, err = dbus.SessionBus()
	if err != nil {
		panic(err)
	} else if reply, err := conn.RequestName(WATCHER_SERVICE, dbus.NameFlagDoNotQueue); err != nil {
		panic(err)
	} else if reply != dbus.RequestNameReplyPrimaryOwner {
		panic(errors.New(WATCHER_SERVICE + " taken"))
	} else if reply, err = conn.RequestName(HOST_SERVICE, dbus.NameFlagDoNotQueue); err != nil {
		panic(err)
	} else if reply != dbus.RequestNameReplyPrimaryOwner {
		panic(errors.New(HOST_SERVICE + " taken"))
	}

	// Put StatusNotifierWatcher object up
	conn.ExportMethodTable(
		map[string]interface{}{
			"RegisterStatusNotifierItem":   addItem,
			"UnregisterStatusNotifierItem": func(string, dbus.Sender) {}, // We dont care, see monitorItem
		},
		WATCHER_PATH,
		WATCHER_INTERFACE,
	)

	// Add Introspectable interface
	conn.Export(introspect.Introspectable(INTROSPECT_XML), WATCHER_PATH, dbuscall.INTROSPECT_INTERFACE)

	// Add properties interface
	watcherProperties = prop.New(
		conn,
		WATCHER_PATH,
		map[string]map[string]*prop.Prop{
			WATCHER_INTERFACE: {
				"IsStatusNotifierHostRegistered": {true, false, prop.EmitTrue, nil},
				"ProtocolVersion":                {0, false, prop.EmitTrue, nil},
				"RegisteredStatusItems":          {[]string{}, false, prop.EmitTrue, nil},
			},
		},
	)
}


type EventType int

const (
	ItemCreated EventType = iota
	ItemRemoved
	ItemUpdated
	MenuUpdated
)

type Event struct {
	eventType EventType
	sender    string
	path      dbus.ObjectPath // Menupath if eventType is MenuUpdated, ItemPath otherwise
}

var events = make(chan Event)



func updateWatcherProperties() {
	ids := make([]string, 0, len(itemCollection.items))
	for _, item := range itemCollection.items {
		ids = append(ids, item.sender+":"+string(item.itemPath))
	}
	watcherProperties.Set(WATCHER_INTERFACE, "RegisteredStatusItems", dbus.MakeVariant(ids))
}





func updateItem(item *Item) {
	props := dbuscall.GetAllProps(conn, item.sender, item.itemPath, ITEM_INTERFACE)

	item.Id = getStringOr(props, "ID", "")
	item.Category = getStringOr(props, "Category", "")
	item.Status = getStringOr(props, "Status", "")
	item.IconName = getStringOr(props, "IconName", "")
	item.IconAccessibleDesc = getStringOr(props, "IconAccessibleDesc", "")
	item.AttentionIconName = getStringOr(props, "AttentionIconName", "")
	item.AttentionAccessibleDesc = getStringOr(props, "AttentionAccessibleDesc", "")
	item.Title = getStringOr(props, "Title", "")
	item.menuPath = getDbusPath(props, "Menu")
	item.iconThemePath = getStringOr(props, "IconThemePath", "")

	if item.IconName == "" {
		item.IconName = collectPixMap(props, "IconPixmap")
	} else if item.iconThemePath != "" {
		image.CopyIcons(item.IconName, item.iconThemePath)
	}
	if item.AttentionIconName == "" {
		item.AttentionIconName = collectPixMap(props, "AttentionIconPixmap")
	} else if item.iconThemePath != "" {
		image.CopyIcons(item.AttentionIconName, item.iconThemePath)
	}
}

func fetchMenu(item *Item) {
	item.Menu, item.menuIds = []MenuItem{}, []string{}
	obj := conn.Object(item.sender, item.menuPath)
	if call := obj.Call(MENU_INTERFACE+".GetLayout", dbus.Flags(0), 0, -1, []string{}); call.Err != nil {
		log.Println(call.Err)
	} else if len(call.Body) < 2 {
		log.Println("Retrieved", len(call.Body), "arguments, expected 2")
	} else if _, ok := call.Body[0].(uint32); !ok {
		log.Println("Expected uint32 as first return argument, got:", reflect.TypeOf(call.Body[0]))
	} else if interfaces, ok := call.Body[1].([]interface{}); !ok {
		log.Println("Expected []interface{} as second return argument, got:", reflect.TypeOf(call.Body[1]))
	} else if menu, menuIds, err := parseMenu(interfaces); err != nil {
		log.Println("Error retrieving menu", err)
	} else if len(menu.SubMenus) > 0 {
		item.Menu, item.menuIds = menu.SubMenus, slice.Remove(menuIds, menu.Id)
	} else {
		item.Menu, item.menuIds = []MenuItem{menu}, menuIds
	}
}


func getStringOr(m map[string]dbus.Variant, key string, fallback string) string {
	if variant, ok := m[key]; ok {
		if res, ok := variant.Value().(string); !ok {
			log.Println("Looking for string at", key, "got:", reflect.TypeOf(variant.Value()))
		} else {
			return res
		}
	}

	return fallback
}

func getBoolOr(m map[string]dbus.Variant, key string, fallback bool) bool {
	if variant, ok := m[key]; ok {
		if res, ok := variant.Value().(bool); !ok {
			log.Println("Looking for boolean at", key, "got:", reflect.TypeOf(variant.Value()))
		} else {
			return res
		}
	}

	return fallback
}

func getInt32Or(m map[string]dbus.Variant, key string, fallback int32) int32 {
	if variant, ok := m[key]; ok {
		if res, ok := variant.Value().(int32); !ok {
			log.Println("Looking for int at", key, "got:", reflect.TypeOf(variant.Value()))
		} else {
			return res
		}
	}
	return fallback
}

func getDbusPath(m map[string]dbus.Variant, key string) dbus.ObjectPath {
	if variant, ok := m[key]; ok {
		if res, ok := variant.Value().(dbus.ObjectPath); !ok {
			log.Println("Looking for ObjectPath at", key, "got:", reflect.TypeOf(variant.Value()))
		} else {
			return res
		}
	}
	return ""
}

func parseMenu(value []interface{}) (MenuItem, []string, error) {
	var menuItem = MenuItem{}
	var id int32
	var ok bool
	var m map[string]dbus.Variant
	var s []dbus.Variant

	if len(value) < 3 {
		return MenuItem{}, []string{}, errors.New("Wrong length")
	} else if id, ok = value[0].(int32); !ok {
		return MenuItem{}, []string{}, errors.New("Expected int32, got: " + reflect.TypeOf(value[0]).String())
	} else if m, ok = value[1].(map[string]dbus.Variant); !ok {
		return MenuItem{}, []string{}, errors.New("Excpected dbus.Variant, got: " + reflect.TypeOf(value[1]).String())
	} else if s, ok = value[2].([]dbus.Variant); !ok {
		return MenuItem{}, []string{}, errors.New("expected []dbus.Variant, got: " + reflect.TypeOf(value[2]).String())
	}

	menuItem.Id = fmt.Sprintf("%d", id)

	if menuItem.Type = getStringOr(m, "type", "standard");
		!slice.Among(menuItem.Type, "standard", "separator") {
		return MenuItem{}, []string{}, errors.New("Illegal menuitem type: " + menuItem.Type)
	}
	menuItem.Label = getStringOr(m, "label", "")
	menuItem.Enabled = getBoolOr(m, "enabled", true)
	menuItem.Visible = getBoolOr(m, "visible", true)
	if menuItem.IconName = getStringOr(m, "icon-name", ""); menuItem.IconName == "" {
		// FIXME: Look for pixmap
	}
	if menuItem.ToggleType = getStringOr(m, "toggle-type", ""); !slice.Among(menuItem.ToggleType, "checkmark", "radio", "") {
		return MenuItem{}, []string{}, errors.New("Illegal toggle-type: " + menuItem.ToggleType)
	}

	menuItem.ToggleState = getInt32Or(m, "toggle-state", -1)
	var menuIds = []string{menuItem.Id}
	if childrenDisplay := getStringOr(m, "children-display", ""); childrenDisplay == "submenu" {
		for _, variant := range s {
			if interfaces, ok := variant.Value().([]interface{}); !ok {
				return MenuItem{}, []string{}, errors.New("Submenu item not of type []interface")
			} else {
				if submenuItem, subIds, err := parseMenu(interfaces); err != nil {
					return MenuItem{}, []string{}, err
				} else {
					menuIds = append(menuIds, subIds...)
					menuItem.SubMenus = append(menuItem.SubMenus, submenuItem)
				}
			}
		}
	} else if childrenDisplay != "" {
		log.Println("warning: ignoring unknown children-display type:", childrenDisplay)
	}

	return menuItem, menuIds, nil
}



func collectPixMap(m map[string]dbus.Variant, key string) string {
	if variant, ok := m[key]; ok {
		if arrs, ok := variant.Value().([][]interface{}); !ok {
			log.Println("Looking for [][]interface{} at"+key+", got:", reflect.TypeOf(variant.Value()))
		} else {
			res := make(image.Icon, 0)
			for _, arr := range arrs {
				for len(arr) > 2 {
					width := arr[0].(int32)
					height := arr[1].(int32)
					pixels := arr[2].([]byte)
					res = append(res, image.Img{Width: width, Height: height, Pixels: pixels})
					arr = arr[3:]
				}
			}
			return image.SaveAsPngToSessionIconDir(res)
		}
	}
	return ""
}

