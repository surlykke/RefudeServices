/*
 * Copyright (c) 2017 Christian Surlykke
 *
 * This file is part of the RefudeServices project.
 * It is distributed under the GPL v2 license.
 * Please refer to the GPL2 file for a copy of the license.
 */

package main

import (

	"os"
	"github.com/rakyll/magicmime"
	"log"
	"fmt"
	"net/http"
	"net"
	"context"
	"encoding/json"
	"io/ioutil"
	"bytes"
	"os/exec"
	"regexp"
	"github.com/surlykke/RefudeServices/xdg"
	"path/filepath"
)

type MimeType struct {
	DefaultApplications []string
}

var client = http.Client {
	Transport: &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", xdg.RuntimeDir() + "/org.refude.desktop-service")
		},
	},
}

func getJson(path string, res interface{}) error {
	url := "http://localhost" + path
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	if err = json.Unmarshal(body, res); err != nil {
		return err
	}
	return nil
}

func postJson(path string, data interface{}) error {
	url := "http://localhost" + path
	fmt.Println("Posting against: ", url)
	if byteArr, err := json.Marshal(data); err != nil {
		return err
	} else if request,err := http.NewRequest("POST", url, bytes.NewReader(byteArr)); err != nil {
		return err
	} else if response,err := client.Do(request); err != nil {
		return err
	} else {
		defer response.Body.Close()
		fmt.Println(response.Status)
		return nil
	}
}

func getDefaultApp(mimetypeid string) (string, error) {
	fmt.Println("Looking for ", mimetypeid)
	mimetype := MimeType{}
	if err := getJson("/mimetypes/" + mimetypeid, &mimetype); err != nil {
		return "", err
	} else if len(mimetype.DefaultApplications) > 0 {
		return mimetype.DefaultApplications[0], nil
	} else {
		return "", nil
	}
}

var schemePattern = func() *regexp.Regexp {
	pattern, err := regexp.Compile(`^(\w+):.*$`)
	if err != nil {
		panic(err)
	}
	return pattern
}()

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: RefudeXdgOpen { file | URL}")
	}

	arg := os.Args[1]

	var mimetypeId = ""

	match := schemePattern.FindStringSubmatch(arg)
	if match != nil {
		mimetypeId = "x-scheme-handler/" + match[1]
	} else {
		var err error
		if arg, err = filepath.Abs(arg); err != nil {
			log.Fatal(err)
		}

		if err := magicmime.Open(magicmime.MAGIC_MIME_TYPE | magicmime.MAGIC_SYMLINK | magicmime.MAGIC_ERROR); err != nil {
			log.Fatal(err)
		}

		defer magicmime.Close()
		mimetypeId, _ = magicmime.TypeByFile(arg)
	}

	log.Println("mimetypeId: ", mimetypeId)

	if len(mimetypeId) == 0 {
		log.Fatal("Could not determine type of " + arg)
	} else if app,err := getDefaultApp(mimetypeId); err != nil {
		log.Fatal("Error querying default app of ", mimetypeId, err)
	} else if len(app) > 0 {
		path := "/applications/" + app
		payload := struct{ Arguments []string }{ Arguments: []string{arg}}
		if err = postJson(path, &payload); err != nil {
			log.Fatal("Error launching " + string(app[0]) + " with " + arg)
		}
	} else {
		fmt.Println("Calling refudeAppChooser ", arg, mimetypeId)
		exec.Command("refudeAppChooser", arg, mimetypeId).Start()
	}

}
