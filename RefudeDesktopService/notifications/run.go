// Copyright (c) 2017,2018 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package notifications

import (
	"sort"
	"time"

	"github.com/surlykke/RefudeServices/lib/resource"
)

var notifications = make(map[uint32]*Notification)

var notificationsMap = resource.MakeResourceMap()
var Notifications = resource.MakeServer(notificationsMap)

var removals = make(chan removal)
var reaper = make(chan uint32)

func Run() {
	var updates = make(chan *Notification)
	go DoDBus(updates, removals)

	updateCollections()
	for {
		select {
		case notification := <-updates:
			notifications[notification.Id] = notification
			notificationsMap.Set(notificationSelf(notification.Id), resource.MakeJsonResouceWithEtag(notification))
			updateCollections()
		case rem := <-removals:
			var path = string(notificationSelf(rem.id))
			if _, ok := notifications[rem.id]; !ok {
				continue
			} else {
				delete(notifications, rem.id)
				notificationsMap.Remove(path)
				updateCollections()
				conn.Emit(NOTIFICATIONS_PATH, NOTIFICATIONS_INTERFACE+".NotificationClosed", rem.id, rem.reason)
			}
		case id := <-reaper:
			if n, ok := notifications[id]; ok {
				var now = time.Now()
				var age = now.Sub(n.Created)
				if age < time.Hour {
					time.AfterFunc(time.Minute*61-age, func() {
						reaper <- n.Id
					})
				} else {
					delete(notifications, id)
					conn.Emit(NOTIFICATIONS_PATH, NOTIFICATIONS_INTERFACE+".NotificationClosed", id, Expired)
				}
				updateCollections()
			}
		}
		notificationsMap.Broadcast()
	}
}

func updateCollections() {
	var all = make(resource.Selfielist, 0, len(notifications))
	for _, notification := range notifications {
		all = append(all, notification)
	}
	sort.Sort(all)
	notificationsMap.Set("/notifications", resource.MakeJsonResouceWithEtag(all))
	notificationsMap.Set("/notifications/brief", resource.MakeJsonResouceWithEtag(all.GetSelfs()))

}

func notificationIsExpired(res interface{}) bool {
	n, ok := res.(*Notification)
	if !ok {
		return false
	}
	return !time.Now().Before(n.Expires)
}
