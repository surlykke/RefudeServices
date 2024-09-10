// Copyright (c) Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
package main

import (
	"net/http"
	"strings"

	"github.com/surlykke/RefudeServices/applications"
	"github.com/surlykke/RefudeServices/browsertabs"
	"github.com/surlykke/RefudeServices/desktop"
	"github.com/surlykke/RefudeServices/file"
	"github.com/surlykke/RefudeServices/icons"
	"github.com/surlykke/RefudeServices/lib/log"
	"github.com/surlykke/RefudeServices/lib/repo"
	"github.com/surlykke/RefudeServices/lib/requests"
	"github.com/surlykke/RefudeServices/lib/respond"
	"github.com/surlykke/RefudeServices/notifications"
	"github.com/surlykke/RefudeServices/options"
	"github.com/surlykke/RefudeServices/ping"
	"github.com/surlykke/RefudeServices/power"
	"github.com/surlykke/RefudeServices/start"
	"github.com/surlykke/RefudeServices/statusnotifications"
	"github.com/surlykke/RefudeServices/watch"
	"github.com/surlykke/RefudeServices/wayland"

	_ "net/http/pprof"
)

func main() {
	var opts = options.GetOpts()
	go icons.Run()

	go wayland.Run(opts.IgnoreWinAppIds)
	go applications.Run()

	if !opts.NoNotifications {
		log.Info("Notifications enabled")
		go notifications.Run()
	} else {
		log.Info("Notifications disabled")
	}

	go power.Run()

	if !opts.NoTray {
		log.Info("Tray enabled")
		go statusnotifications.Run()
	} else {
		log.Info("Tray disabled")
	}

	go start.Run()

	http.Handle("/ping", ping.WebsocketHandler)
	http.HandleFunc("/tabsink", browsertabs.ServeHTTP)
	http.HandleFunc("/icon", icons.ServeHTTP)
	http.HandleFunc("/desktop/", desktop.ServeHTTP)
	http.HandleFunc("/watch", watch.ServeHTTP)

	http.HandleFunc("/complete", complete)
	http.HandleFunc("/search", search)
	http.HandleFunc("/file/", file.ServeHTTP)
	http.HandleFunc("/flash", notifications.ServeFlash)
	http.HandleFunc("/", repo.ServeHTTP)

	if err := http.ListenAndServe(":7938", nil); err != nil {
		log.Warn("http.ListenAndServe failed:", err)
	}
}

func search(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		var term = strings.ToLower(requests.GetSingleQueryParameter(r, "term", ""))
		var fromPath = requests.GetSingleQueryParameter(r, "from", "/start")
		if from := repo.GetUntyped(fromPath); from != nil {
			respond.AsJson(w, from.Search(term))
		} else {
			respond.NotFound(w)
		}
	} else {
		respond.NotAllowed(w)
	}
}

func complete(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		var prefix = requests.GetSingleQueryParameter(r, "prefix", "")
		var paths = make([]string, 0, 1000)
		for _, p := range []string{"/flash", "/icon?name=", "/desktop/", "/complete?prefix=", "/search?", "/watch"} {
			if strings.HasPrefix(p, prefix) {
				paths = append(paths, p)
			}
		}

		for _, res := range repo.GetListUntyped(prefix) {
			paths = append(paths, res.GetPath())
		}

		respond.AsJson(w, paths)
	} else {
		respond.NotAllowed(w)
	}
}
