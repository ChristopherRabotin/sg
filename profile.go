package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/jmcvetta/randutil"
	"io/ioutil"
	"math"
	"strconv"
	"strings"
	"time"
)

// duration allows automatic unmarshaling of a duration from XML.
type duration struct {
	time.Duration
}

// UnmarshalXMLAttr unmarshals a duration.
func (dur *duration) UnmarshalXMLAttr(attr xml.Attr) (err error) {
	parsed, err := time.ParseDuration(attr.Value)
	if err != nil {
		return
	}
	*dur = duration{parsed}
	return nil
}

// Profile stores the whole test profile
type Profile struct {
	Name  string        `xml:"name,attr"`
	UID   string        `xml:"uid,attr"`
	Tests []*StressTest `xml:"test"`
}

// StressTest stores the one stress test.
type StressTest struct {
	Name        string        `xml:"name,attr"`
	Description string        `xml:"description"`
	CriticalTh  duration      `xml:"critical,attr"`
	WarningTh   duration      `xml:"warning,attr"`
	Requests    []*RequestXML `xml:"request"`
}

// RequestXML stores the request as XML.
// It is kept in XML until it is executed to read from the parent response as needed.
type RequestXML struct {
	Parent   *RequestXML
	Children *[]RequestXML `xml:"request"`
	URL      *URL          `xml:"url"`
	Headers  *Tokenized    `xml:"headers"`
	Data     *Tokenized    `xml:"data"`
}

// Todo: function to generate a URL from this struct.
type URL struct {
	Base   string      `xml:"base,attr"`
	Tokens *[]URLToken `xml:"token"`
}

// Validate confirms the validity of a URL.
func (u *URL) Validate() {
	for _, tok := range *u.Tokens {
		tok.Validate()
		if !strings.Contains(u.Base, tok.Token) {
			panic(fmt.Errorf("cannot find token %s in base %s.", tok.Token, u.Base))
		}
	}
}

// Generate returns a new URL based on the base and the tokens.
func (u *URL) Generate() (url string) {
	url = u.Base
	if u.Tokens != nil {
		for _, tok := range *u.Tokens {
			url = strings.Replace(url, tok.Token, tok.Generate(), -1)
		}
	}
	return
}

// URLToken handles the generate of tokens for the URL.
type URLToken struct {
	Token     string `xml:"token,attr"`
	Choices   string `xml:"choices,attr"`
	Pattern   string `xml:"pattern,attr"`
	MinLength int    `xml:"min,attr"`
	MaxLength int    `xml:"max,attr"`
}

// Validate checks that the definition of this token is met, and panics otherwise.
func (t *URLToken) Validate() {
	if t.Token == "" {
		panic("empty token in URL definition")
	}
	if t.Choices == "" && t.Pattern == "" {
		panic("URL Token is missing both Choices and Pattern")
	}
	if t.Choices != "" {
		if !strings.Contains(t.Choices, "|") {
			panic(fmt.Errorf("choices %s does not contain any separator (|)", t.Choices))
		}
		if t.MinLength != 0 || t.MaxLength != 0 {
			log.Warning("min and max definitions have no effect in URL Tokens of type Choice")
		}
	}
	if t.Pattern != "" {
		if t.Pattern != "num" && t.Pattern != "alpha" && t.Pattern != "alphanum" {
			panic(fmt.Errorf("unknown pattern %s in URL Token", t.Pattern))
		}
		if t.MinLength < 0 || t.MaxLength < 0 {
			panic("min or max is negative in URL Token")
		}
		if t.MinLength > t.MaxLength {
			panic("min definition is greater than max definition in URL Token")
		}
	}
}

// Generate returns a new value for a token according to the definition.
func (t *URLToken) Generate() (r string) {
	if t.Choices != "" {
		r, _ = randutil.ChoiceString(strings.Split(t.Choices, "|"))
	}
	switch t.Pattern {
	case "alpha":
		r, _ = randutil.StringRange(t.MinLength, t.MaxLength, randutil.Alphabet)
	case "alphanum":
		r, _ = randutil.AlphaStringRange(t.MinLength, t.MaxLength)
	case "num":
		rInt, _ := randutil.IntRange(int(math.Pow10(t.MinLength)), int(math.Pow10(t.MinLength)))
		r = strconv.FormatInt(int64(rInt), 10)
	}
	return
}

// Todo: Tokenized helping functions.
type Tokenized struct {
	Response string `xml:"responseToken,attr"`
	Header   string `xml:"headerToken,attr"`
}

func loadProfile(profileFile string) (*Profile, error) {
	if profileFile == "" {
		return nil, errors.New("profile filename is empty")
	}
	profileData, err := ioutil.ReadFile(profileFile)
	if err != nil {
		return nil, fmt.Errorf("error loading profile %s: %s\n", profileFile, err)
	}
	profile := Profile{}
	if err = xml.Unmarshal(profileData, &profile); err != nil {
		return nil, fmt.Errorf("error loading profile %s: %s\n", profileFile, err)
	}
	// Let's set the parent requests on all children.
	for _, test := range profile.Tests {
		for _, request := range test.Requests {
			setParentRequest(nil, request.Children)
		}
	}
	return &profile, nil
}

func setParentRequest(parent *RequestXML, children *[]RequestXML) {
	if children != nil {
		for _, child := range *children {
			child.Parent = parent
			setParentRequest(&child, child.Children)
		}
	}
}
