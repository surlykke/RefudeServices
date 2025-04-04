package browser

import (
	"strings"

	"github.com/surlykke/RefudeServices/lib/entity"
	"github.com/surlykke/RefudeServices/lib/response"
)

type Tab struct {
	entity.Base
	Id        string
	BrowserId string
	Url       string
}

func (this *Tab) DoPost(action string) response.Response {
	commands <- tabCommand{BrowserName: this.BrowserId, Operation: "focus", TabId: this.Id}
	return response.Accepted()
}

func (this *Tab) DoDelete() response.Response {
	commands <- tabCommand{BrowserName: this.BrowserId, Operation: "delete", TabId: this.Id}
	return response.Accepted()
}

func (this *Tab) OmitFromSearch() bool {
	return strings.HasPrefix(this.Url, "http://localhost:7938/desktop")
}
