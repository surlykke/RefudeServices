// Copyright (c) 2017 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package power

import (
	"fmt"
	"github.com/godbus/dbus"
	"github.com/surlykke/RefudeServices/lib/dbusutils"
	"github.com/surlykke/RefudeServices/lib/resource"
	"strings"
)

const UPowService = "org.freedesktop.UPower"
const UPowPath = "/org/freedesktop/UPower"
const UPowerInterface = "org.freedesktop.UPower"
const DisplayDevicePath = "/org/freedesktop/UPower/devices/DisplayDevice"
const UPowerDeviceInterface = "org.freedesktop.UPower.Device"
const login1Service = "org.freedesktop.login1"
const login1Path = "/org/freedesktop/login1"
const managerInterface = "org.freedesktop.login1.Manager"

func Run(updateStream chan <- resource.Update) {

	// Get on the bus
	signals := make(chan *dbus.Signal, 100)
	dbusConn.Signal(signals)
	dbusConn.BusObject().Call(
		"org.freedesktop.DBus.AddMatch",
		0,
		"type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged', sender='org.freedesktop.UPower'")

	updateStream <- resource.Update{Mappings: []resource.Mapping{{"/session", MapPowerActions()}}}

	var devices = make(map[dbus.ObjectPath]*Device)

	enumCall := dbusConn.Object(UPowService, UPowPath).Call(UPowerInterface+".EnumerateDevices", dbus.Flags(0))
	devicePaths := append(enumCall.Body[0].([]dbus.ObjectPath), DisplayDevicePath)
	for _, path := range devicePaths {
		var device = &Device{}

		device.AbstractResource = resource.MakeAbstractResource(resource.Standardizef("/devices%s", path[strings.LastIndex(string(path), "/"):]), DeviceMediaType)
		devices[path] = device
		updateDevice(device, dbuscall.GetAllProps(dbusConn, UPowService, path, UPowerDeviceInterface))
		updateStream <- resource.Update{Mappings: []resource.Mapping{{device.GetSelf(), device}}}
	}

	for signal := range signals {
		fmt.Println("Signal: ", signal)
		if signal.Name == "org.freedesktop.DBus.Properties.PropertiesChanged" {
			if device, ok := devices[signal.Path]; ok {
				var copy = *device
				// Brute force here, we update all, as I've seen some problems with getting out of sync after suspend..
				updateDevice(&copy, dbuscall.GetAllProps(dbusConn, UPowService, signal.Path, UPowerDeviceInterface))
				updateStream <- resource.Update{Mappings: []resource.Mapping{{copy.GetSelf(), &copy}}}
			}
			// TODO Handle device added/removed
			// (need hardware to test)
		}
	}
}

var dbusConn = func() *dbus.Conn {
	if conn, err := dbus.SystemBus(); err != nil {
		panic(err)
	} else {
		return conn
	}
}()

var possibleActionValues = map[string][]string{
	"PowerOff":    {"Shutdown", "Power off the machine", "system-shutdown"},
	"Reboot":      {"Reboot", "Reboot the machine", "system-reboot"},
	"Suspend":     {"Suspend", "Suspend the machine", "system-suspend"},
	"Hibernate":   {"Hibernate", "Put the machine into hibernation", "system-suspend-hibernate"},
	"HybridSleep": {"HybridSleep", "Put the machine into hybrid sleep", "system-suspend-hibernate"}}

func MapPowerActions() *Session {
	var session Session
	session.AbstractResource = resource.MakeAbstractResource("/session", SessionMediaType);

	for id, pv := range possibleActionValues {
		if "yes" == dbusConn.Object(login1Service, login1Path).Call(managerInterface+".Can"+id, dbus.Flags(0)).Body[0].(string) {
			var dbusEndPoint = managerInterface + "." + id
			var executer = func() {
				fmt.Println("Calling", login1Service, login1Path, managerInterface+"."+id)
				dbusConn.Object(login1Service, login1Path).Call(dbusEndPoint, dbus.Flags(0), false)
			}
			session.ResourceActions[id] = resource.ResourceAction{Description: pv[1], IconName: pv[2], Executer: executer}
		}
	}

	return &session
}

func updateDevice(d *Device, m map[string]dbus.Variant) {
	for key, variant := range m {
		switch key {
		case "NativePath":
			d.NativePath = variant.Value().(string)
		case "Vendor":
			d.Vendor = variant.Value().(string)
		case "Model":
			d.Model = variant.Value().(string)
		case "Serial":
			d.Serial = variant.Value().(string)
		case "UpdateTime":
			d.UpdateTime = variant.Value().(uint64)
		case "Type":
			d.Type = deviceType(variant.Value().(uint32))
		case "PowerSupply":
			d.PowerSupply = variant.Value().(bool)
		case "HasHistory":
			d.HasHistory = variant.Value().(bool)
		case "HasStatistics":
			d.HasStatistics = variant.Value().(bool)
		case "Online":
			d.Online = variant.Value().(bool)
		case "Energy":
			d.Energy = variant.Value().(float64)
		case "EnergyEmpty":
			d.EnergyEmpty = variant.Value().(float64)
		case "EnergyFull":
			d.EnergyFull = variant.Value().(float64)
		case "EnergyFullDesign":
			d.EnergyFullDesign = variant.Value().(float64)
		case "EnergyRate":
			d.EnergyRate = variant.Value().(float64)
		case "Voltage":
			d.Voltage = variant.Value().(float64)
		case "TimeToEmpty":
			d.TimeToEmpty = variant.Value().(int64)
		case "TimeToFull":
			d.TimeToFull = variant.Value().(int64)
		case "Percentage":
			d.Percentage = variant.Value().(float64)
		case "IsPresent":
			d.IsPresent = variant.Value().(bool)
		case "State":
			d.State = deviceState(variant.Value().(uint32))
		case "IsRechargeable":
			d.IsRechargeable = variant.Value().(bool)
		case "Capacity":
			d.Capacity = variant.Value().(float64)
		case "Technology":
			d.Technology = deviceTecnology(variant.Value().(uint32))
		}
	}
}