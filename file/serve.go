// Copyright (c) Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package file

import (
	"github.com/surlykke/RefudeServices/lib/log"
)

func Get(filePath string) *File {
	if file, err := makeFile(filePath); err != nil {
		log.Warn("Could not make file from", filePath, err)
		return nil
	} else if file == nil {
		return nil
	} else {
		return file
	}
}
