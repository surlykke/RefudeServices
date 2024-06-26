// Copyright (c) Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package searchutils

import (
	"strings"
)

func Match(term, name string, keywords ...string) int {
	name = strings.ToLower(name)
	if rnk := strings.Index(name, term); rnk > -1 {
		return rnk
	}
	for _, keyword := range keywords {
		if strings.Index(strings.ToLower(keyword), term) > -1 {
			return 1000
		}
	}
	return -1
}
