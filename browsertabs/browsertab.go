package browsertabs

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/surlykke/RefudeServices/lib/icon"
	"github.com/surlykke/RefudeServices/lib/mediatype"
	"github.com/surlykke/RefudeServices/lib/path"
	"github.com/surlykke/RefudeServices/lib/repo"
	"github.com/surlykke/RefudeServices/lib/resource"
	"github.com/surlykke/RefudeServices/lib/respond"
	"github.com/surlykke/RefudeServices/watch"
)

type Tab struct {
	resource.ResourceData
	Url string
}

func (this *Tab) Id() string {
	return string(this.Path[len("/tab/"):])
}

func (this *Tab) DoPost(w http.ResponseWriter, r *http.Request) {
	watch.Publish("focusTab", this.Id())
	respond.Accepted(w)
}

func (this *Tab) DoDelete(w http.ResponseWriter, r *http.Request) {
	watch.Publish("closeTab", this.Id())
	respond.Accepted(w)
}

func (this *Tab) OmitFromSearch() bool {
	return strings.HasPrefix(this.Url, "http://localhost:7938/desktop")
}

// This is the api that the browserextensions use
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if r.Body == nil {
			respond.UnprocessableEntity(w, errors.New("No data"))
		} else if bytes, err := io.ReadAll(r.Body); err != nil {
			respond.ServerError(w, err)
		} else {
			var data = make([]map[string]string, 30)
			if err := json.Unmarshal(bytes, &data); err != nil {
				respond.UnprocessableEntity(w, err)
			} else {
				var tabs = make([]resource.Resource, 0, len(data))
				for _, d := range data {
					var title = d["title"]
					if len(title) > 60 { // Shorten title a bit
						if spacePos := strings.Index(title[60:], " "); spacePos > -1 {
							title = title[0:60+spacePos] + "..."
						} else {
							title = title[0:60] + "..."
						}
					}
					var url = d["url"]
					if queryStart := strings.Index(url, "?"); queryStart > -1 {
						url = url[0:queryStart+1] + "..."
					} else if len(url) > 60 {
						url = url[0:60] + "..."
					}
					var iconUrl = d["favIcon"]
					var tab = &Tab{ResourceData: *resource.MakeBase(path.Of("/tab/", d["id"]), title, "", icon.Name(iconUrl), mediatype.Tab)}
					tab.Url = url
					tab.AddAction("focus", title, "Focus tab", icon.Name(iconUrl))
					//tab.AddDeleteAction("close", title, "Close tab", "")

					tabs = append(tabs, tab)
				}
				respond.Ok(w)
				repo.Replace(tabs, "/tab/")
				watch.Publish("search", "")
			}
		}
	} else {
		respond.NotAllowed(w)
	}
}
