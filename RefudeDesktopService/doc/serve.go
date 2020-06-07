package doc

import (
	"net/http"

	. "github.com/surlykke/RefudeServices/lib/i18n"
	"github.com/surlykke/RefudeServices/lib/respond"
	"github.com/surlykke/RefudeServices/lib/xdg"
)

type DocType struct {
	Type       string
	Name       string
	NamePlural string
	Collection string
	Doc        string
}

var types = map[string]DocType{
	"application": {
		Type:       "application",
		Name:       Tr("Application"),
		NamePlural: Tr("Applications"),
		Collection: "/applications",
		Doc:        "TODO",
	},
	"mimetype": {
		Type:       "mimetype",
		Name:       Tr("Mimetype"),
		NamePlural: Tr("Mimetypes"),
		Collection: "/mimetypes",
		Doc:        "TODO",
	},
	"window": {
		Type:       "window",
		Name:       Tr("Window"),
		NamePlural: Tr("Windows"),
		Collection: "/windows",
		Doc:        "TODO",
	},
	"file": {
		Type:       "file",
		Name:       Tr("File"),
		NamePlural: Tr("Files"),
		Collection: "/file",
		Doc:        "TODO",
	},
	"icontheme": {
		Type:       "icontheme",
		Name:       Tr("Icontheme"),
		NamePlural: Tr("Iconthemes"),
		Collection: "/iconthemes",
		Doc:        "TODO",
	},
	"notification": {
		Type:       "notification",
		Name:       Tr("Notification"),
		NamePlural: Tr("Notifications"),
		Collection: "/notifications",
		Doc:        "TODO",
	},
	"device": {
		Type:       "device",
		Name:       Tr("Device"),
		NamePlural: Tr("Devices"),
		Collection: "/devices",
		Doc:        "TODO",
	},
	"session_action": {
		Type:       "session_action",
		Name:       Tr("Leave action"),
		NamePlural: Tr("Leave actions"),
		Collection: "/session/actions",
		Doc:        "TODO",
	},
}

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/doc/refudeservices.md" {
		var path = xdg.DataHome + "/RefudeServices/assets/README.md"
		http.ServeFile(w, r, path)
	} else if r.URL.Path == "/doc/types" {
		respond.AsJson(w, r, &respond.StandardFormat{
			Self:  "/doc/types",
			Type:  "doctypes",
			Title: "Refude document types",
			Data:  types,
		})
	} else {
		respond.NotFound(w)
	}
}
