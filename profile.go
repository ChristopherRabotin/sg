package main

import (
	"encoding/xml"
	"fmt"
	"github.com/jmcvetta/randutil"
	"math"
	"strconv"
	"strings"
	"time"
)

// duration allows automatic unmarshaling of a duration from XML.
type duration struct {
	time.Duration
}

func (dur *duration) UnmarshalXML(d *xml.Decoder, el xml.StartElement) (err error) {
	var v string
	d.DecodeElement(&v, &el)
	parsed, err := time.ParseDuration(v)
	if err != nil {
		return
	}
	*dur = duration{parsed}
	return
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
	CriticalTh  duration      `xml:"gauge>critical"`
	WarningTh   duration      `xml:"gauge>warning"`
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

// Get returns a new URL based on the base and the tokens.
func (u *URL) Get() (url string) {
	url = u.Base
	if u.Tokens != nil {
		for _, tok := range *u.Tokens {
			url = strings.Replace(url, tok.Token, tok.Generate(), -1)
		}
	}
	return
}

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
