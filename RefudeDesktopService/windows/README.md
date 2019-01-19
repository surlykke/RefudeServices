# RefudeWmService

RefudeWmService is a part of the [RefudeServices](http://github.com/surlykke/RefudeServices) project. 
It exposes information about the display and open windows (a small subset of EWMH).

## Standard Resources 

- [`/ping`](http://github.com/surlykke/RefudeServices#ping)
- [`/doc`](http://github.com/surlykke/RefudeServices#doc) This document
- [`/windows/<id>`](): Information about open windows
- [`/icons/<id>`](#icons): Window icons

## Display

The display resource resides on `/display` and shows position and resolution of the display. Example:

```
RefudeGET wm-service /display
```
could yield:

```
{
	"W":1920,
	"H":1080,
	"Screens":[]
}
```
This signifies that current display is 1920 by 1080 pixels.

FIXME: screens

## Windows

Windows resources are of the form `/windows/<id>`. Each resource represents an open window. Example:

```
RefudeGET wm-service /windows/
```

```json
{
    "Id": 48234503,
    "X": 117,
    "Y": 0,
    "H": 971,
    "W": 1803,
    "Name": "Vim",
    "IconUrl": "../icons/15171C54F27D7692",
    "States": [

    ],
    "Actions": {
        "_default": {
            "Name": "Vim",
            "Comment": "Raise and focus"
        }
    },
    "RelevanceHint": -3
}
```

The fields are:
- `id`: The window id. Same as the id X-Windows provides for the window
- `X`, `Y`: Position of windows upper left corner  
- `H`, `W`: Height and width of window
- `Name`: Window title
- `IconUrl`: Relative path of the window icon. Resolves relative to the path of the window resource, so in this example, you'd find
  the icon on `/icons/15171C54F27D7692`
- `States`: Window states as defined in [EWMH](https://specifications.freedesktop.org/wm-spec/wm-spec-latest.html#idm140200472615568).
- `Actions`: Window actions. Only one action is offered: `_default` which raises and focuses the window
- `RelevanceHint`: Minus stacking order. The window on top of the windowstack has value 0, next -1 and so on

## Icons

Serves window icons in png format. Try for example:

`RefudeGET wm-service /windows/`

to get a list of your open windows. Get one of them, say:

`RefudeGET wm-service /windows/46137349`

Resolve the IconUrl relative to the window url, to do something like:

`RefudeGET wm-service /icons/7BABF63233A8D30F | feh -`

(substitute any image-viewing application for 'feh' as you wish)

