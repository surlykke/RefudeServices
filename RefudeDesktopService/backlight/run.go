package backlight

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/surlykke/RefudeServices/lib/searchutils"

	"github.com/fsnotify/fsnotify"
	"github.com/surlykke/RefudeServices/lib/respond"
)

const backlightdir = "/sys/class/backlight"
const brightness = "/brightness"
const lenbld = len(backlightdir)
const lenbn = len(brightness)

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if backlight, ok := devices.Load().(DeviceMap)[r.URL.Path]; ok {
		if r.Method == "GET" {
			respond.AsJson(w, backlight.ToStandardFormat())
		} else {
			respond.NotAllowed(w)
		}
	} else {
		respond.NotFound(w)
	}
}

func SearchBacklights(collector *searchutils.Collector) {
	for _, device := range devices.Load().(DeviceMap) {
		collector.Collect(device.ToStandardFormat())
	}
}

func AllPaths() []string {
	var deviceMap = devices.Load().(DeviceMap)
	var paths = make([]string, 0, len(deviceMap))
	for path, _ := range deviceMap {
		paths = append(paths, path)
	}
	return paths
}

func Run() {
	devices.Store(retrieveDevices())
	for {
		select {
		case _ = <-watcher.Events:
			devices.Store(retrieveDevices())
		case err := <-watcher.Errors:
			fmt.Println("Error from watcher:", err)
		}
	}
}

func init() {
	devices.Store(make(DeviceMap))
}

var watcher = func() *fsnotify.Watcher {
	if tmp, err := fsnotify.NewWatcher(); err != nil {
		panic(err)
	} else if err := tmp.Add(backlightdir); err != nil {
		panic(fmt.Errorf("Unable to watch %s: %s", backlightdir, err))
	} else {
		return tmp
	}
}()

var watchedDirs = map[string]bool{}

func retrieveDevices() DeviceMap {
	for subdir, _ := range watchedDirs {
		watcher.Remove(backlightdir + "/" + subdir + "/brightness")
	}
	if deviceDirs, err := filepath.Glob(backlightdir + "/*"); err != nil {
		log.Println("Error globbing for backlight catalogues: ", err)
		return make(DeviceMap)
	} else {
		var devMap = make(DeviceMap)
		for _, deviceDir := range deviceDirs {
			var dev = &Device{Id: deviceDir[len(backlightdir)+1:]}
			var maxBrightnessPath = deviceDir + "/max_brightness"
			var brightnessPath = deviceDir + "/brightness"
			if err := watcher.Add(brightnessPath); err != nil {
				fmt.Println("Unable to watch", brightnessPath, err)
			} else if dev.maxBrightness, err = readUint64(maxBrightnessPath); err != nil {
				fmt.Println("Unable to read", maxBrightnessPath, err)
				watcher.Remove(brightnessPath)
			} else if dev.brightness, err = readBrightness(brightnessPath, dev.maxBrightness); err != nil {
				fmt.Println("Problem reading brightness:", brightnessPath, err)
				watcher.Remove(brightnessPath)
			}
			dev.BrightnessPct = uint8(100 * dev.brightness / dev.maxBrightness)
			dev.Updated = time.Now()
			fmt.Println(">>>>>>>>>>>>>>>>> Setting: ", "/backlight/"+dev.Id)
			devMap["/backlight/"+dev.Id] = dev
		}
		return devMap
	}
}

func readBrightness(brightnessPath string, maxBrightness uint64) (uint64, error) {
	if brightness, err := readUint64(brightnessPath); err != nil {
		return 0, err
	} else if brightness > maxBrightness {
		return 0, fmt.Errorf("brightness > max_brightnes: %d %d", brightness, maxBrightness)
	} else {
		return brightness, nil
	}
}

func readUint64(filepath string) (uint64, error) {
	if bytes, err := ioutil.ReadFile(filepath); err != nil {
		return 0, err
	} else if val, err := strconv.ParseUint(strings.TrimSpace(string(bytes)), 10, 64); err != nil {
		return 0, err
	} else {
		return val, nil
	}
}

var devices atomic.Value