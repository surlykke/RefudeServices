package xlib

/**
 * All communication with xlib (and X) happens through this package
 */

/*
#cgo LDFLAGS: -lX11
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <X11/Xlib.h>
// Cant use 'type' in Go, hence...
inline int getType(XEvent* e) { return e->type; }

// Using C macros in Go seems tricky, so..
inline int ds(Display* d) { return DefaultScreen(d); }
inline Window rw(Display *d, int screen) { return RootWindow(d, screen); }

// Accessing a field inside a union inside a struct from Go is messy. Hence these helpers
inline XConfigureEvent* xconfigure(XEvent* e) { return &(e->xconfigure); }
inline XPropertyEvent* xproperty(XEvent* e) { return &(e->xproperty); }

// Converting sequences unsigned chars to byte or long. Most easily done in C, so..
const unsigned long sizeOfLong = sizeof(long);
inline char getByte(unsigned char* data, int index) { return ((char*)data)[index]; }
inline long getLong(unsigned char* data, int index) { return ((long*)data)[index]; }

XEvent createClientMessage32(Window window, Atom message_type, long l0, long l1, long l2, long l3, long l4) {
	XEvent event;
	memset(&event, 0, sizeof(XEvent));
	event.xclient.type = ClientMessage;
	event.xclient.serial = 0;
	event.xclient.send_event = 1;
	event.xclient.message_type = message_type;
	event.xclient.window = window;
	event.xclient.format = 32;
	event.xclient.data.l[0] = l0;
	event.xclient.data.l[1] = l1;
	event.xclient.data.l[2] = l2;
	event.xclient.data.l[3] = l3;
	event.xclient.data.l[4] = l4;
	return event;
}

int forgiving_X_error_handler(Display *d, XErrorEvent *e)
{
	char errorMsg[80];
	XGetErrorText(d, e->error_code, errorMsg, 80);
	printf("Got error: %s\n", errorMsg);
	return 0;
}
*/
import "C"
import (
	"errors"
	"fmt"
	"log"
	"unsafe"
)

func init() {
	C.XSetErrorHandler(C.XErrorHandler(C.forgiving_X_error_handler))
}

/**
 * Wrapper around connections to X11. Not threadsafe, so users must make sure only
 * one go-routine at a time uses an instance of this.
 */
type Connection struct {
	display    *C.Display
	rootWindow 	C.Window

	atomCache map[string]C.Atom
	atomNameCache map[C.Atom]string
}

// Either 'Property' or X,Y,W,H will be set
type Event struct {
	Window   uint32
	Property string
	X,Y,W,H  int
}

func MakeConnection() *Connection {
	var conn = Connection{}
	conn.display = C.XOpenDisplay(nil)
	var defaultScreen = C.ds(conn.display)
	conn.rootWindow = C.rw(conn.display, defaultScreen)
	conn.atomCache = make(map[string]C.Atom)
	conn.atomNameCache = make(map[C.Atom]string)
	return &conn
}

func (c *Connection) Listen(window uint32) {
	if window == 0 {
		C.XSelectInput(c.display, c.rootWindow, C.SubstructureNotifyMask|C.PropertyChangeMask)
	} else {
		C.XSelectInput(c.display, C.ulong(window), C.PropertyChangeMask)
	}
}


// Will hang until either a property change or a configure event happens
func (c *Connection)NextEvent() (Event, error) {
	var event C.XEvent
	for {
		if err := CheckError(C.XNextEvent(c.display, &event)); err != nil {
			return Event{}, err
		} else {
			switch C.getType(&event) {
			case C.PropertyNotify:
				var xproperty = C.xproperty(&event)
				return Event{Window: uint32(xproperty.window), Property: c.atomName(xproperty.atom)}, nil
			case C.ConfigureNotify:
				var xconfigure = C.xconfigure(&event)
				return Event{Window: uint32(xconfigure.window),
				             X: int(xconfigure.x), Y: int(xconfigure.y), W: int(xconfigure.width), H: int(xconfigure.height)}, nil
			}
		}
	}
}


func (c *Connection) atom(name string) C.Atom {
	if val, ok := c.atomCache[name]; ok {
		return val
	} else {
		var cName = C.CString(name)
		defer C.free(unsafe.Pointer(cName))
		val = C.XInternAtom(c.display, cName, 1)
		if val == C.None {
			log.Fatal(fmt.Sprintf("Atom %s does not exist", name))
		}
		c.atomCache[name] = val
		return val
	}

}


func (c *Connection) atomName(atom C.Atom) string {
	if name, ok := c.atomNameCache[atom]; ok {
		return name
	} else {
		var tmp = C.XGetAtomName(c.display, atom)
		defer C.XFree(unsafe.Pointer(tmp))
		c.atomNameCache[atom] = C.GoString(tmp)
		return c.atomNameCache[atom]
	}
}


func (c *Connection) GetBytes(window uint32, property string) ([]byte, error) {
	var ulong_window = C.ulong(window)
	if ulong_window == 0 {
		ulong_window = c.rootWindow
	}
	var prop = c.atom(property)
	var long_offset C.long
	var long_length = C.long(256)

	var result []byte
	var actual_type_return C.Atom
	var actual_format_return C.int
	var nitems_return C.ulong
	var bytes_after_return C.ulong
	var prop_return *C.uchar
	for {
		var status = C.XGetWindowProperty(c.display, ulong_window, prop, long_offset, long_length, 0, C.AnyPropertyType,
			&actual_type_return, &actual_format_return, &nitems_return, &bytes_after_return, &prop_return)

		if err := CheckError(status); err != nil {
			return nil, err;
		} else if actual_format_return != 8 {
			return nil, errors.New(fmt.Sprintf("Expected format 8, got %d", actual_format_return))
		}

		var currentLen = len(result);
		var growBy = int(nitems_return)
		var neededCapacity = currentLen + growBy

		if cap(result) < neededCapacity {
			tmp := make([]byte, currentLen, neededCapacity)
			for i := 0; i < currentLen; i++ {
				tmp[i] = result[i]
			}
			result = tmp
		}

		for i := 0; i < growBy; i++ {
			result = append(result, byte(C.getByte(prop_return, C.int(i))))
		}

		C.XFree(unsafe.Pointer(prop_return))

		if (bytes_after_return == 0) {
			return result, nil
		}
		long_length = C.long(bytes_after_return)/4 + 1
		long_offset = long_offset + C.long(nitems_return)*4
	}
}

func (c *Connection) GetPropStr(wId uint32, property string) (string, error) {
	bytes, err := c.GetBytes(wId, property)
	return string(bytes), err
}

func (c *Connection) GetUint32s(window uint32, property string) ([]uint32, error) {
	var ulong_window = C.ulong(window)
	if ulong_window == 0 {
		ulong_window = c.rootWindow
	}
	var prop = c.atom(property)
	var long_offset C.long
	var long_length = C.long(256)

	var result []uint32
	var actual_type_return C.Atom
	var actual_format_return C.int
	var nitems_return C.ulong
	var bytes_after_return C.ulong
	var prop_return *C.uchar
	for {
		var error = C.XGetWindowProperty(c.display, ulong_window, prop, long_offset, long_length, 0, C.AnyPropertyType,
			&actual_type_return, &actual_format_return, &nitems_return, &bytes_after_return, &prop_return)

		if err := CheckError(error); err != nil {
			return nil, err;
		} else if actual_format_return != 32 {
			return nil, errors.New(fmt.Sprintf("Expected format 32, got %d", actual_format_return))
		}

		var currentLen = len(result);
		var growBy = int(nitems_return)
		var neededCapacity = currentLen + growBy

		if cap(result) < neededCapacity {
			tmp := make([]uint32, currentLen, neededCapacity)
			for i := 0; i < currentLen; i++ {
				tmp[i] = result[i]
			}
			result = tmp
		}

		for i := 0; i < growBy; i++ {
			result = append(result, uint32(C.getLong(prop_return, C.int(i))))
		}

		C.XFree(unsafe.Pointer(prop_return))

		if (bytes_after_return == 0) {
			return result, nil
		}

		long_length = C.long(bytes_after_return)/4 + 1
		long_offset = long_offset + C.long(nitems_return);
	}
}

func (c *Connection) GetAtoms(wId uint32, property string) ([]string, error) {
	if atoms, err := c.GetUint32s(wId, property); err != nil {
		return nil, err
	} else {
		var states = make([]string, len(atoms), len(atoms))
		for i, atom := range atoms {
			states[i] = c.atomName(C.ulong(atom))
		}
		return states, nil
	}
}


func (c *Connection) GetParent(wId uint32) (uint32, error) {
	var root_return C.ulong
	var parent_return C.ulong
	var children_return *C.ulong
	var nchildren_return C.uint
	for {
		if C.XQueryTree(c.display, C.ulong(wId), &root_return, &parent_return, &children_return, &nchildren_return) == 0 {
			return 0, errors.New("Error from XQueryTree")
		} else {
			if children_return != nil {
				C.XFree(unsafe.Pointer(children_return))
			}
			if parent_return == c.rootWindow {
				return wId, nil
			} else {
				wId = uint32(parent_return)
			}
		}
	}
}

func (c *Connection) GetGeometry(wId uint32) (int, int, int, int, error) {
	return 0, 0, 0, 0, nil
}


func CheckError(error C.int) error {
	switch error {
	case 0:
		return nil;
	case C.BadAlloc:
		return errors.New("The server failed to allocate the requested resource or server memory.")
	case C.BadAtom:
		return errors.New("A value for an Atom argument does not name a defined Atom.")
	case C.BadValue:
		return errors.New("Some numeric value falls outside the range of values accepted by the request. Unless a specific range is specified for an argument, the full range defined by the argument's type is accepted. Any argument defined as a set of alternatives can generate this error.")
	case C.BadWindow:
		return errors.New(fmt.Sprintf("A value for a Window argument does not name a defined Window"))
	default:
		return errors.New(fmt.Sprintf("Uknown error: %d", error))
	}
}


func (c *Connection) RaiseAndFocusWindow(wId uint32) {
	var event = C.createClientMessage32(C.Window(wId), c.atom("_NET_ACTIVE_WINDOW"), 2, 0, 0, 0, 0);
	var mask C.long = C.SubstructureRedirectMask | C.SubstructureNotifyMask
	C.XSendEvent(c.display, c.rootWindow, 0, mask, &event);
	C.XFlush(c.display)
}