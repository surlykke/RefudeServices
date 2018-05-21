// Copyright (c) 2017,2018 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package main

import (
	"github.com/surlykke/RefudeServices/lib/service"
	"fmt"
	"net/http"
	"github.com/surlykke/RefudeServices/lib/requestutils"
	"github.com/surlykke/RefudeServices/lib/resource"
)

var removals = make(chan removal)
var notifications = make(map[uint32]*Notification)

func Run() {
	var updates = make(chan *Notification)
	go DoDBus(updates, removals)

	for {
		select {
		case notification := <-updates:
			notifications[notification.Id] = notification
			var actions = notification.getActions()
			service.Unmap(fmt.Sprintf("/notifications/%d", notification.Id))
			service.RemoveAll(fmt.Sprintf("/actions/%d", notification.Id))
			for _, action := range actions {
				resource.Relate(&action.AbstractResource, &notification.AbstractResource)
				service.Map(action)
			}
			service.Map(notification)

		case rem := <-removals:
			fmt.Println("Got removal..")
			if notification, ok := notifications[rem.id]; ok {
				if rem.internalId == 0 || rem.internalId == notification.internalId {
					service.Unmap(fmt.Sprintf("/notifications/%d", rem.id))
					service.RemoveAll(fmt.Sprintf("/actions/%d", rem.id))
					delete(notifications, rem.id)
					notificationClosed(rem.id, rem.reason)
				}
			}
		}
	}
}

func path(id uint32) string {
	return fmt.Sprintf("/notifications/%d", id)
}

func (n *Notification) POST(w http.ResponseWriter, r *http.Request) {
	action := requestutils.GetSingleQueryParameter(r, "action", "default")
	conn.Emit(NOTIFICATIONS_PATH, NOTIFICATIONS_INTERFACE+".ActionInvoked", n.Id, action)
	w.WriteHeader(http.StatusAccepted)
}

func (n *Notification) DELETE(w http.ResponseWriter, r *http.Request) {
	removals <- removal{n.Id, 0, Dismissed}
	w.WriteHeader(http.StatusAccepted)
}

