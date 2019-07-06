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
	"time"

	"github.com/surlykke/RefudeServices/lib/resource"
)

type Notification struct {
	resource.Links
	Id        uint32
	Sender    string
	Subject   string
	Body      string
	IconName  string `json:",omitempty"`
	Image     string `json:",omitempty"`
	imagePath string
	Created   time.Time
	Expires   time.Time `json:",omitempty"`
}

type NotificationImage struct {
	imagePath string
}

func (ni *NotificationImage) GetSelf() string {
	return ""
}

func (ni *NotificationImage) GetEtag() string {
	return ""
}

func (ni *NotificationImage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if "GET" != r.Method {
		w.WriteHeader(http.StatusMethodNotAllowed)
	} else {
		fmt.Println("Serving", ni.imagePath)
		http.ServeFile(w, r, ni.imagePath)
	}
}

func notificationSelf(id uint32) string {
	return fmt.Sprintf("/notification/%d", id)
}
