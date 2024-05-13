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
	"github.com/surlykke/RefudeServices/config"
	"github.com/surlykke/RefudeServices/desktop"
	"github.com/surlykke/RefudeServices/file"
	"github.com/surlykke/RefudeServices/icons"
	"github.com/surlykke/RefudeServices/lib/log"
	"github.com/surlykke/RefudeServices/lib/repo"
	"github.com/surlykke/RefudeServices/lib/requests"
	"github.com/surlykke/RefudeServices/lib/resource"
	"github.com/surlykke/RefudeServices/lib/respond"
	"github.com/surlykke/RefudeServices/notifications"
	"github.com/surlykke/RefudeServices/ping"
	"github.com/surlykke/RefudeServices/power"
	"github.com/surlykke/RefudeServices/start"
	"github.com/surlykke/RefudeServices/statusnotifications"
	"github.com/surlykke/RefudeServices/watch"
	"github.com/surlykke/RefudeServices/wayland"

	_ "net/http/pprof"
)

func main() {
	log.Info("Running")

	go start.Run()
	go icons.Run()
	go file.Run()

	go wayland.Run()
	go applications.Run()
	go browsertabs.Run()
	if config.Notifications.Enabled {
		go notifications.Run()
	}
	go power.Run()
	go statusnotifications.Run()

	http.Handle("/ping", ping.WebsocketHandler)
	http.HandleFunc("/tabsink", browsertabs.ServeHTTP)
	http.HandleFunc("/icon", icons.ServeHTTP)
	http.HandleFunc("/desktop/", desktop.ServeHTTP)
	http.HandleFunc("/watch", watch.ServeHTTP)

	http.HandleFunc("/complete", complete)
	http.HandleFunc("/search", search)
	http.HandleFunc("/", serveResources)

	if err := http.ListenAndServe(":7938", nil); err != nil {
		log.Warn("http.ListenAndServe failed:", err)
	}
}

func serveResources(w http.ResponseWriter, r *http.Request) {
	var path = r.URL.Path
	if strings.HasSuffix(path, "/") {
		resource.ServeList(w, r, repo.FindList(path))
	} else {
		if res := repo.FindSingle(path); res != nil {
			resource.ServeSingleResource(w, r, res)
		} else {
			respond.NotFound(w)
		}
	}
}

func search(w http.ResponseWriter, r *http.Request) {
	var term = strings.ToLower(requests.GetSingleQueryParameter(r, "term", ""))
	if r.URL.Query().Has("from") {
		if from := repo.FindSingle(r.URL.Query().Get("from")); from != nil {
			if searcable, ok := from.(resource.Searchable); ok {
				resource.ServeList(w, r, searcable.Search(term))
				return 
			}
		}
		respond.NotFound(w)
	} else {
		resource.ServeList(w, r, repo.DoSearch(term))
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

		for _, res := range repo.FindList(prefix) {
			paths = append(paths, res.Data().Path)
		}

		respond.AsJson(w, paths)
	} else {
		respond.NotAllowed(w)
	}
}

