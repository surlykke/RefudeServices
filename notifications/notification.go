// Copyright (c) Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
package notifications

import (
	"net/http"
	"strconv"
	"time"

	"github.com/surlykke/RefudeServices/lib/requests"
	"github.com/surlykke/RefudeServices/lib/resource"
	"github.com/surlykke/RefudeServices/lib/respond"
)

type Urgency uint8

const (
	Low      Urgency = 0
	Normal           = 1
	Critical         = 2
)

var (
	LowBytes      = []byte(`"low"`)
	NormalBytes   = []byte(`"normal"`)
	CriticalBytes = []byte(`"critical"`)
)

func (u Urgency) MarshalJSON() ([]byte, error) {
	switch u {
	case Low:
		return LowBytes, nil
	case Normal:
		return NormalBytes, nil
	case Critical:
		return CriticalBytes, nil
	default:
		panic("unknown urgency")
	}
}

type UnixTime time.Time // Behaves like Time, but json-marshalls to milliseconds since epoch

func (ut UnixTime) MarshalJSON() ([]byte, error) {
	var buf = make([]byte, 0, 22)
	buf = strconv.AppendInt(buf, time.Time(ut).UnixMilli(), 10)
	return buf, nil
}

type Notification struct {
	resource.ResourceData
	NotificationId uint32
	Sender         string
	Created        time.Time
	Expires        time.Time
	Deleted        bool
	Urgency        Urgency
	NActions       map[string]string `json:"actions"`
	Hints          map[string]interface{}
	iconName       string
	IconSize       uint32 `json:",omitempty"`
}

func (n *Notification) NotExpired() bool {
	return time.Now().Before(time.Time(n.Expires))
}

func (n *Notification) DoPost(w http.ResponseWriter, r *http.Request) {
	var action = requests.GetSingleQueryParameter(r, "action", "default")

	if _, ok := n.NActions[action]; ok {
		if err := conn.Emit(NOTIFICATIONS_PATH, NOTIFICATIONS_INTERFACE+".ActionInvoked", n.NotificationId, action); err != nil {
			respond.ServerError(w, err)
		} else {
			respond.Accepted(w)
		}
	} else {
		respond.NotFound(w)
	}
}

func (n *Notification) DoDelete(w http.ResponseWriter, r *http.Request) {
	removals <- notificationRemoval{id: n.NotificationId, reason: Dismissed}	
	respond.Accepted(w)
}
