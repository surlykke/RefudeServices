// Copyright (c) 2017,2018,2019 Christian Surlykke
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
	"github.com/surlykke/RefudeServices/icons"
	dbuscall "github.com/surlykke/RefudeServices/lib/dbusutils"
	"github.com/surlykke/RefudeServices/lib/respond"
)

const UPowService = "org.freedesktop.UPower"
const UPowPath = "/org/freedesktop/UPower"
const UPowerInterface = "org.freedesktop.UPower"
const DevicePrefix = "/org/freedesktop/UPower/devices"
const DisplayDevicePath = dbus.ObjectPath(DevicePrefix + "/DisplayDevice")
const UPowerDeviceInterface = "org.freedesktop.UPower.Device"

func subscribe() chan *dbus.Signal {
	var signals = make(chan *dbus.Signal, 100)

	dbusConn.Signal(signals)
	dbusConn.BusObject().Call(
		"org.freedesktop.DBus.AddMatch",
		0,
		"type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged', sender='org.freedesktop.UPower'")

	dbusConn.BusObject().Call(
		"org.freedesktop.DBus.AddMatch",
		0,
		"type='signal',interface='org.freedesktop.UPower',member='DeviceAdded', sender='org.freedesktop.UPower'")

	dbusConn.BusObject().Call(
		"org.freedesktop.DBus.AddMatch",
		0,
		"type='signal',interface='org.freedesktop.UPower',member='DeviceRemoved', sender='org.freedesktop.UPower'")

	return signals
}

func retrieveDevicePaths() []dbus.ObjectPath {
	enumCall := dbusConn.Object(UPowService, UPowPath).Call(UPowerInterface+".EnumerateDevices", dbus.Flags(0))
	return append(enumCall.Body[0].([]dbus.ObjectPath), DisplayDevicePath)
}

func retrieveDevice(path dbus.ObjectPath) *Device {
	var device = Device{}
	device.DisplayDevice = path == DisplayDevicePath
	device.DbusPath = path
	var lastSlash = strings.LastIndex(string(path), "/")
	var title = strings.Title(strings.Join(strings.Split(string(path)[lastSlash+1:], "_"), " "))
	updateDevice(&device, dbuscall.GetAllProps(dbusConn, UPowService, path, UPowerDeviceInterface))
	var self string
	if strings.HasPrefix(string(device.DbusPath), DevicePrefix) {
		self = fmt.Sprintf("/device%s", device.DbusPath[len(DevicePrefix):])
	} else {
		self = fmt.Sprintf("/device%s", device.DbusPath)
	}
	device.Resource = respond.MakeResource(self, title, icons.IconUrl(device.IconName), &device, "device")
	return &device
}

var dbusConn = func() *dbus.Conn {
	if conn, err := dbus.SystemBus(); err != nil {
		panic(err)
	} else {
		return conn
	}
}()

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
			d.Percentage = int8(variant.Value().(float64))
		case "IsPresent":
			d.IsPresent = variant.Value().(bool)
		case "State":
			d.State = deviceState(variant.Value().(uint32))
		case "IconName":
			d.IconName = variant.Value().(string)
		case "IsRechargeable":
			d.IsRechargeable = variant.Value().(bool)
		case "Capacity":
			d.Capacity = variant.Value().(float64)
		case "Technology":
			d.Technology = deviceTecnology(variant.Value().(uint32))
		}
	}
}
