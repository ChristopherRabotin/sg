package main

import (
	"encoding/xml"
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
	Tests []*StressTest `xml:"sg>test"`
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
	Base string `xml:"base,attr"`
}

type URLToken struct {
	Token     string `xml:"token,attr"`
	Choices   string `xml:choices,attr"`
	Pattern   string `xml:pattern,attr"`
	MinLength int    `xml:min,attr"`
	MaxLength int    `xml:max,attr"`
}
// Todo: Tokenized helping functions.
type Tokenized struct {
	Response string `xml:"responseToken,attr"`
	Header   string `xml:"headerToken,attr"`
}
