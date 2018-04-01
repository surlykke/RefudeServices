// Copyright (c) 2017 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package main

import (
	"github.com/surlykke/RefudeServices/lib/service"
	"fmt"
)

func main() {
	// Initially, put an empty list of notifications up, signalling we don't have any yet
	fmt.Println("Createing empty /notifications/")
	service.MkDir("/notifications")
	Setup()
	service.SetSearchFunction(searchFunction)
	service.Serve("org.refude.notifications-service")
}
