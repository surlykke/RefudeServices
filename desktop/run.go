// Copyright (c) Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
package desktop

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"

	"github.com/surlykke/RefudeServices/lib/link"
	"github.com/surlykke/RefudeServices/lib/log"
	"github.com/surlykke/RefudeServices/lib/relation"
	"github.com/surlykke/RefudeServices/lib/repo"
	"github.com/surlykke/RefudeServices/lib/requests"
	"github.com/surlykke/RefudeServices/lib/resource"
	"github.com/surlykke/RefudeServices/lib/respond"
	"github.com/surlykke/RefudeServices/lib/stringhash"
	"github.com/surlykke/RefudeServices/watch"
	"github.com/surlykke/RefudeServices/wayland"
)

//go:embed html
var sources embed.FS

var bodyTemplate *template.Template
var StaticServer http.Handler

func init() {
	var bytes []byte
	var err error

	if bytes, err = sources.ReadFile("html/bodyTemplate.html"); err != nil {
		log.Panic(err)
	}

	bodyTemplate = template.Must(template.New("bodyTemplate").Parse(string(bytes)))
}

type row struct {
	//	Heading  string
	Class    string
	IconUrl  string
	Title    string
	Comment  string
	Href     string
	Relation relation.Relation
	Profile  string
}

func headingRow(heading string) row {
	return row{Title: heading, Class: "heading"}
}

func actionRow(action link.Link) row {
	return row{IconUrl: action.IconUrl, Title: action.Title, Href: action.Href, Relation: action.Relation, Class: "selectable"}
}

func resourceRow(sr resource.Resource) row {
	var comment string
	if sr.Data().Comment != "" {
		comment = sr.Data().Profile + ": " + sr.Data().Comment
	}
	return row{IconUrl: sr.Data().IconUrl, Title: sr.Data().Title, Comment: comment, Href: sr.Data().Path, Relation: relation.Self, Profile: sr.Data().Profile, Class: "selectable"}
}

func init() {
	var tmp http.Handler

	if projectDir, ok := os.LookupEnv("DEV_PROJECT_ROOT_DIR"); ok {
		// Used when developing
		tmp = http.FileServer(http.Dir(projectDir + "/desktop/html"))
	} else if htmlDir, err := fs.Sub(sources, "html"); err == nil {
		// Otherwise, what's baked in
		tmp = http.FileServer(http.FS(htmlDir))
	} else {
		log.Panic(err)
	}
	StaticServer = http.StripPrefix("/desktop", tmp)

}

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/desktop/body":
		if r.Method != "GET" {
			respond.NotAllowed(w)
		} else {
			var resourcePath = requests.GetSingleQueryParameter(r, "resource", "/start")
			if res := repo.FindSingle(resourcePath); res != nil {
				var (
					term           = requests.GetSingleQueryParameter(r, "search", "")
					actions        = res.Data().ActionLinks(term)
					sf, searchable = res.(resource.Searchable)
					subresources   = []resource.Resource{}
					rows           = make([]row, 0, len(actions)+len(subresources)+2)
				)
				if searchable {
					subresources = sf.Search(term)
				}
				if len(actions) > 0 {
					rows = append(rows, headingRow("Actions"))
				}
				for _, a := range actions {
					rows = append(rows, actionRow(a))
				}
				if len(actions) > 0 && len(subresources) > 0 {
					rows = append(rows, headingRow("Related"))
				}
				for _, sr := range subresources {
					rows = append(rows, resourceRow(sr))
				}

				var m = map[string]any{
					"Searchable": searchable,
					"Title":      res.Data().Title,
					"Icon":       res.Data().IconUrl,
					"Term":       term,
					"Rows":       rows,
				}
				var etag = buildETag(res.Data().Title, res.Data().IconUrl, rows)
				if r.Header.Get("if-none-match") == etag {
					respond.NotModified(w)
					return
				}
				w.Header().Set("ETag", etag)

				if err := bodyTemplate.Execute(w, m); err != nil {
					log.Warn("Error executing bodyTemplate:", err)
				}

			} else {
				respond.NotFound(w)
			}
		}
	case "/desktop/show":
		if r.Method != "POST" {
			respond.NotAllowed(w)
		} else {
			wayland.RememberActive()
			watch.Publish("showDesktop", "")
			respond.Accepted(w)
		}
	case "/desktop/hash":
		if r.Method == "GET" {
			// Go Json cannot handle uint64, so we convert to string
			respond.AsJson(w, "")
		} else {
			respond.NotAllowed(w)
		}
	case "/desktop/hide":
		if r.Method != "POST" {
			respond.NotAllowed(w)
		} else {
			switch requests.GetSingleQueryParameter(r, "restore", "") {
			case "window":
				wayland.ActivateRememberedActive()
				fallthrough
			case "tab":
				watch.Publish("restoreTab", "")
			}
			respond.Accepted(w)
		}
	case "/desktop/bodyTemplate.html":
		respond.NotFound(w)
	default:
		StaticServer.ServeHTTP(w, r)
	}
}


func buildETag(title, icon string, rows []row) string {
	var hash uint64 = 0
	hash = stringhash.FNV1a(title, icon)
	for _, row := range rows {
		hash = hash ^ stringhash.FNV1a(row.Title, row.Comment, row.Href, row.IconUrl, row.Profile, string(row.Relation))
	}
	return fmt.Sprintf(`"%X"`, hash)
}
