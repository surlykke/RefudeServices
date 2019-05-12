// Copyright (c) 2017,2018,2019 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package resource

// For embedding
type Links struct {
	Self       string `json:"_self,omitempty"`
	RefudeType string `json:"_refudetype,omitempty"`
}

func (l *Links) GetSelf() string {
	return l.Self
}
