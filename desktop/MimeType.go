package main

import (
	"encoding/xml"
	"github.com/surlykke/RefudeServices/common"
	"io/ioutil"
	"fmt"
	"regexp"
	"strings"
	"github.com/surlykke/RefudeServices/resources"
)

const freedesktopOrgXml = "/usr/share/mime/packages/freedesktop.org.xml"

type MimeType struct {
	resources.FallbackHandler
	Type                   string
	Subtype                string
	Comment                string
	Acronym                string
	ExpandedAcronym        string
	Aliases                common.StringSet
	Globs                  common.StringSet
	SubClassOf             common.StringSet
	Icon                   string
	GenericIcon            string
	AssociatedApplications common.StringSet
	DefaultApplications    []string
}

func CollectMimeTypes() map[string]MimeType {
	xmlCollector := struct {
		XMLName   xml.Name `xml:"mime-info"`
		MimeTypes []struct {
			Type    string `xml:"type,attr"`
			Comment []struct {
				Lang string `xml:"lang,attr"`
				Text string `xml:",chardata"`
			} `xml:"comment"`
			Acronym         string `xml:"acronym"`
			ExpandedAcronym string `xml:"expanded-acronym"`
			Alias           []struct {
				Type string `xml:"type,attr"`
			} `xml:"alias"`
			Glob []struct {
				Pattern string `xml:"pattern,attr"`
			} `xml:"glob"`
			SubClassOf []struct {
				Type string `xml:"type,attr"`
			} `xml:"sub-class-of"`
			Icon struct {
				Name string `xml:"name,attr"`
			} `xml:"icon"`
			GenericIcon struct {
				Name string `xml:"name,attr"`
			} `xml:"generic-icon"`
		} `xml:"mime-type"`
	}{}

	xmlInput, err := ioutil.ReadFile(freedesktopOrgXml)
	if err != nil {
		fmt.Println("Unable to open ", freedesktopOrgXml, ": ", err)
	}
	parseErr := xml.Unmarshal(xmlInput, &xmlCollector)
	if parseErr != nil {
		fmt.Println("Error parsing: ", parseErr)
	}

	typePattern, err := regexp.Compile(`^([^/]+)/([^/]+)$`)
	if err != nil {
		panic(err)
	}

	res := make(map[string]MimeType)
	for _, tmp := range xmlCollector.MimeTypes {
		mimeType := MimeType{}

		typeElements := typePattern.FindStringSubmatch(tmp.Type)
		if len(typeElements) == 3 {
			mimeType.Type = typeElements[1]
			mimeType.Subtype = typeElements[2]
		} else {
			fmt.Println("Incomprehensible mimetype: ", tmp.Type)
			continue
		}

		for _, tmpComment := range tmp.Comment {
			if tmpComment.Lang == "" {
				mimeType.Comment = tmpComment.Text // FIXME
			}
		}

		mimeType.Acronym = tmp.Acronym
		mimeType.ExpandedAcronym = tmp.ExpandedAcronym

		mimeType.Aliases = make(common.StringSet)
		for _, aliasStruct := range tmp.Alias {
			mimeType.Aliases[aliasStruct.Type] = true
		}

		mimeType.Globs = make(common.StringSet)
		for _, tmpGlob := range tmp.Glob {
			mimeType.Globs[tmpGlob.Pattern] = true
		}

		mimeType.SubClassOf = make(common.StringSet)
		for _, tmpSubClassOf := range tmp.SubClassOf {
			mimeType.SubClassOf[tmpSubClassOf.Type] = true
		}

		if tmp.Icon.Name != "" {
			mimeType.Icon = tmp.Icon.Name
		} else {
			mimeType.Icon = strings.Replace(mimeType.Type, "/", "-", -1)
		}

		if tmp.GenericIcon.Name != "" {
			mimeType.GenericIcon = tmp.GenericIcon.Name
		} else {
			mimeType.GenericIcon = mimeType.Type + "-x-generic"
		}

		mimeType.AssociatedApplications = make(common.StringSet)
		mimeType.DefaultApplications = make([]string, 0)

		res[mimeType.Type+"/"+mimeType.Subtype] = mimeType
	}

	return res

}
