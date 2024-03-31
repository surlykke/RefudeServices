// Copyright (c) Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
package resource

import (
	"net/http"

	"github.com/surlykke/RefudeServices/lib/link"
	"github.com/surlykke/RefudeServices/lib/relation"
	"github.com/surlykke/RefudeServices/lib/requests"
	"github.com/surlykke/RefudeServices/lib/respond"
	"github.com/surlykke/RefudeServices/lib/searchutils"
)

type Resource interface {
	GetPath() string
	GetTitle() string
	GetComment() string
	GetIconUrl() link.Href
	GetProfile() string
	Actions() link.ActionList
	DeleteAction() (title string, ok bool)
	Links(searchTerm string) link.List
	RelevantForSearch() bool
	GetKeywords() []string
}

type BaseResource struct {
	Path       string
	Title    string    `json:"-"`
	Comment  string    `json:"-"`
	IconUrl  link.Href `json:"-"`
	Profile  string    `json:"-"`
	Keywords []string
}

func (br *BaseResource) GetPath() string {
	return br.Path
}


func (br *BaseResource) GetTitle() string {
	return br.Title
}

func (br *BaseResource) GetComment() string {
	return br.Comment
}

func (br *BaseResource) GetIconUrl() link.Href {
	return br.IconUrl 
}

func (br *BaseResource) GetProfile() string {
	return br.Profile
}

func (br *BaseResource) Actions() link.ActionList {
	return link.ActionList{}
}

func (br *BaseResource) DeleteAction() (string, bool) {
	return "", false
}

func (br *BaseResource) Links(searchTerm string) link.List {
	return link.List{}
}

func (br *BaseResource) RelevantForSearch() bool {
	return true
}

func (br *BaseResource) GetKeywords() []string {
	return br.Keywords
}

func LinkTo(res Resource, context string, rank int) link.Link {
	return link.MakeRanked(res.GetPath(), res.GetTitle(), res.GetIconUrl(), res.GetProfile(), rank)
}

type Postable interface {
	DoPost(w http.ResponseWriter, r *http.Request)
}

type Deleteable interface {
	DoDelete(w http.ResponseWriter, r *http.Request)
}

type ResourceRepo interface {
	GetResources() []Resource
	GetResource(path string) Resource
	Search(term string, threshold int) link.List
}

func SingleResourceServer(res Resource, context string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ServeSingleResource(w, r, res)
	}
}

func ServeSingleResource(w http.ResponseWriter, r *http.Request, res Resource) {
	if r.Method == "GET" {
		var linkSearchTerm = requests.GetSingleQueryParameter(r, "search", "")
		respond.AsJson(w, buildJsonRepresentation(res, linkSearchTerm))
	} else if postable, ok := res.(Postable); ok && r.Method == "POST" {
		postable.DoPost(w, r)
	} else if deletable, ok := res.(Deleteable); ok && r.Method == "DELETE" {
		deletable.DoDelete(w, r)
	} else {
		respond.NotAllowed(w)
	}
}

type jsonRepresentation struct {
	Self    link.Href   `json:"self"`
	Links   link.List   `json:"links"`
	Title   string      `json:"title"`
	Comment string      `json:"comment,omitempty"`
	Icon    link.Href   `json:"icon,omitempty"`
	Profile string      `json:"profile"`
	Data    interface{} `json:"data"`
}

func buildJsonRepresentation(res Resource, searchTerm string) jsonRepresentation {
	var wrapper = jsonRepresentation{}
	wrapper.Self = link.Href(res.GetPath())
	wrapper.Links = buildFilterAndRewriteLinks(res, searchTerm)
	wrapper.Data = res
	wrapper.Title = res.GetTitle()
	wrapper.Comment = res.GetComment()
	wrapper.Icon = res.GetIconUrl()
	wrapper.Profile = res.GetProfile()
	return wrapper
}




func buildFilterAndRewriteLinks(res Resource, searchTerm string) link.List {
	var list = make(link.List, 0, 10)
	for _, action := range res.Actions() {
		var href = res.GetPath()
		if action.Name != "" {
			href += "?action=" + action.Name
		}
		if searchutils.Match(searchTerm, action.Name) < 0 {
			continue
		}
		list = append(list, link.Make(href, action.Title, action.IconUrl, relation.Action))
	}
	if deleteTitle, ok := res.DeleteAction(); ok {
		if searchutils.Match(searchTerm, deleteTitle) > -1 {
			list = append(list, link.Make(res.GetPath(), deleteTitle, "", relation.Delete))
		}
	}

	for _, lnk := range res.Links(searchTerm) {
		list = append(list, lnk)
	}

	return list
}
