// Copyright (c) 2017,2018 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package power

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/surlykke/RefudeServices/lib/respond"
)

type Device struct {
	respond.Resource
	DbusPath         dbus.ObjectPath
	NativePath       string
	Vendor           string
	Model            string
	Serial           string
	UpdateTime       uint64
	Type             string
	IconName         string
	PowerSupply      bool
	HasHistory       bool
	HasStatistics    bool
	Online           bool
	Energy           float64
	EnergyEmpty      float64
	EnergyFull       float64
	EnergyFullDesign float64
	EnergyRate       float64
	Voltage          float64
	TimeToEmpty      int64
	TimeToFull       int64
	Percentage       int8
	IsPresent        bool
	State            string
	IsRechargeable   bool
	Capacity         float64
	Technology       string
	DisplayDevice    bool
}

func Title(d *Device) string {

	// Try to, with the info we have from UPower, make a meaningful Title and Comment
	switch d.Type {
	case "Unknown":
		return "Unknown power device"
	case "Line Power":
		return "Line Power"
	case "Battery":
		return "Battery " + d.Model
	default:
		return d.Model
	}
}

func deviceType(index uint32) string {
	var devType = []string{"Unknown", "Line Power", "Battery", "Ups", "Monitor", "Mouse", "Keyboard", "Pda", "Phone"}
	if index < 0 || index > 8 {
		index = 0
	}
	return devType[index]
}

func deviceState(index uint32) string {
	var devState = []string{"Unknown", "Charging", "Discharging", "Empty", "Fully charged", "Pending charge", "Pending discharge"}
	if index < 0 || index > 6 {
		index = 0
	}
	return devState[index]
}

func deviceTecnology(index uint32) string {
	var devTecnology = []string{"Unknown", "Lithium ion", "Lithium polymer", "Lithium iron phosphate", "Lead acid", "Nickel cadmium", "Nickel metal hydride"}
	if index < 0 || index > 6 {
		index = 0
	}
	return devTecnology[index]
}

func deviceSelf(path dbus.ObjectPath) string {
	if strings.HasPrefix(string(path), DevicePrefix) {
		path = path[len(DevicePrefix):]
	}
	return fmt.Sprintf("/device%s", path)
}
