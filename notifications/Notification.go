// Copyright (c) 2017 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package notifications

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/surlykke/RefudeServices/lib/respond"
)

type Notification struct {
	respond.Links `json:"_links"`
	Id            uint32
	Sender        string
	Subject       string
	Body          string
	Created       time.Time
	Expires       time.Time `json:",omitempty"`
	Actions       map[string]string
	Hints         map[string]interface{}
	self          string
	iconName      string
}

func (n *Notification) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		respond.AsJson(w, n)
	} else if r.Method == "POST" {
		// TODO otheractions
		if n.haveDefaultAction() {
			respond.AcceptedAndThen(w, func() {
				conn.Emit(NOTIFICATIONS_PATH, NOTIFICATIONS_INTERFACE+".ActionInvoked", n.Id, "default")
			})
		} else {
			respond.NotAllowed(w)
		}
	} else if r.Method == "DELETE" {
		respond.AcceptedAndThen(w, func() { removals <- removal{id: n.Id, reason: Dismissed} })

	} else {
		respond.NotAllowed(w)
	}
}

func (n *Notification) haveDefaultAction() bool {
	_, ok := n.Actions["default"]
	return ok
}

func (n *Notification) isUrgent() bool {
	return fmt.Sprintf("%d", n.Hints["urgency"]) == "2"
}

func (n *Notification) isGauge() bool {
	return strings.HasPrefix(fmt.Sprintf("%s", n.Hints["category"]), "x-org.refude.gauge.")
}