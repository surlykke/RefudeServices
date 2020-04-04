// Copyright (c) 2017 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package xdg

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

var commentLine = regexp.MustCompile(`^\s*(#.*)?$`)
var headerLine = regexp.MustCompile(`^\s*\[(.+?)\]\s*`)
var keyValueLine = regexp.MustCompile(`^\s*(..+?)(\[(..+)\])?=(.+)`)
var userDirsLine = regexp.MustCompile(`^\s*(XDG_\w+_DIR)="(.*)"`)

var lcMessage = func() string {
	if os.Getenv("LC_ALL") != "" {
		return os.Getenv("LC_ALL")
	} else if os.Getenv("LC_MESSAGE") != "" {
		return os.Getenv("LC_MESSAGE")
	} else {
		return os.Getenv("LANG")
	}
}()

var lcMessagePattern = regexp.MustCompile(`([^_.@]+)(_[^.@]+)?(\.[^@]+)?(@.*)?`) // 1: language, 2: country, 3: encoding, 4: modifier
var lcMatchers = func() []string {
	if m := lcMessagePattern.FindStringSubmatch(lcMessage); m != nil {
		var lang = m[1]
		var country = m[2]
		var modifier = m[4]

		if country != "" && modifier != "" {
			return []string{
				lang + country + modifier,
				lang + country,
				lang + modifier,
				lang,
			}
		} else if country != "" {
			return []string{
				lang + country,
				lang,
			}
		} else if modifier != "" {
			return []string{
				lang + modifier,
				lang,
			}
		} else {
			return []string{lang}
		}
	} else {
		return []string{}
	}
}()

type Group struct {
	Name    string
	Entries map[string]string
}

type IniFile []*Group

func (inifile IniFile) FindGroup(groupName string) *Group {
	for _, group := range inifile {
		if group.Name == groupName {
			return group
		}
	}
	return nil
}

func ReadIniFile(path string) (IniFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	var iniFile = make(IniFile, 0)
	var currentGroup *Group = nil
	for scanner.Scan() {
		if commentLine.MatchString(scanner.Text()) {
			continue
		} else if m := headerLine.FindStringSubmatch(scanner.Text()); len(m) > 0 {
			if currentGroup = iniFile.FindGroup(m[1]); currentGroup != nil {
				log.Println("Warn: iniFile", path, " has duplicate group entry: ", m[1])
			} else {
				currentGroup = &Group{m[1], make(map[string]string)}
				iniFile = append(iniFile, currentGroup)
			}
		} else if m = keyValueLine.FindStringSubmatch(scanner.Text()); len(m) > 0 {
			if currentGroup == nil {
				return nil, errors.New("Invalid iniFile," + path + ": file must start with a group heading")
			}
			if LocaleMatch(m[3]) || (m[3] == "" && currentGroup.Entries[m[1]] == "") {
				currentGroup.Entries[m[1]] = m[4]
			}
		} else {
			fmt.Println(scanner.Text(), " - not recognized")
		}
	}

	return iniFile, nil
}

func LocaleMatch(loc string) bool {
	for _, lm := range lcMatchers {
		if loc == lm {
			return true
		}
	}
	return false
}

func WriteIniFile(path string, iniFile IniFile) error {
	if file, err := os.Create(path); err != nil {
		return err
	} else {
		defer file.Close()
		for _, group := range iniFile {
			file.WriteString("[" + group.Name + "]\n")
			for key, value := range group.Entries {
				file.WriteString(key + "=" + value + "\n")
			}
		}
		return nil
	}
}

func readUserDirs(home string, configHome string) (map[string]string, error) {
	var res = map[string]string{}
	var file, err = os.Open(configHome + "/user-dirs.dirs")
	if err == nil {
		var scanner = bufio.NewScanner(file)
		for scanner.Scan() {
			if commentLine.MatchString(scanner.Text()) {
				continue
			} else if m := userDirsLine.FindStringSubmatch(scanner.Text()); len(m) > 0 {
				var envVarName = m[1]
				var path = strings.ReplaceAll(m[2], "$HOME", home) // TODO Should we check that path exists, and if not fall back to default?
				res[envVarName] = path
			} else {
				log.Printf("Could not comprehend '%s'", scanner.Text())
			}
		}
	}
	return res, err
}
