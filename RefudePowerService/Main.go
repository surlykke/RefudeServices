/*
 * Copyright (c) 2017 Christian Surlykke
 *
 * This file is part of the RefudeServices project.
 * It is distributed under the GPL v2 license.
 * Please refer to the GPL2 file for a copy of the license.
 */

package main

import (
	"github.com/surlykke/RefudeServices/service"
)

func main() {
	pm := 	&PowerManager{}
	go pm.Run()
	service.Serve("org.refude.power-service")
}