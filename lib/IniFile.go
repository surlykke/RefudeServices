// Copyright (c) 2017 Christian Surlykke
//
// This file is part of the RefudeServices project.
// It is distributed under the GPL v2 license.
// Please refer to the GPL2 file for a copy of the license.
//
package lib

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"

)

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
	var commentLine = regexp.MustCompile(`^\s*(#.*)?$`)
	var headerLine = regexp.MustCompile(`^\s*\[(.+?)\]\s*`)
	var keyValueLine = regexp.MustCompile(`^\s*(..+?)(\[(..+)\])?=(.+)`)

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
	return false; // FIXME
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