package applications

import (
	"fmt"
	"github.com/surlykke/RefudeServices/lib/xdg"
	"golang.org/x/sys/unix"
	"os"
)

var ApplicationsServer = MakeServer(make(DesktopApplicationCollection))
var MimetypesServer = MakeServer(make(MimetypeCollection))

func Run() {
	fd, err := unix.InotifyInit()

	if err != nil {
		panic(err)
	}

	defer unix.Close(fd)

	for _, dataDir := range append(xdg.DataDirs, xdg.DataHome) {
		appDir := dataDir + "/applications"
		if _, err := os.Stat(appDir); os.IsNotExist(err) {
			// path/to/whatever does not exist
		}

		if xdg.DirOrFileExists(appDir) {
			fmt.Println("Watching: " + appDir)
			if _, err := unix.InotifyAddWatch(fd, appDir, unix.IN_CREATE|unix.IN_MODIFY|unix.IN_DELETE); err != nil {
				fmt.Println("Could not watch:", appDir, ":", err)
			}
		}
	}

	if _, err := unix.InotifyAddWatch(fd, xdg.ConfigHome+"/mimeapps.list", unix.IN_CLOSE_WRITE); err != nil {
		panic(err)
	}

	dummy := make([]byte, 100)
	for {
		fmt.Println("Collecting...")
		var mtc MimetypeCollection
		var dac DesktopApplicationCollection
		mtc, dac = Collect();

		MimetypesServer.setResources(mtc)
		ApplicationsServer.setResources(dac)

		if _, err := unix.Read(fd, dummy); err != nil {
			panic(err)
		}
	}
}


