package sg

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/franela/goreq"
	"github.com/jmcvetta/randutil"
	"io/ioutil"
	"math"
	"regexp"
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
	Name        string        `xml:"name,attr"`     // Name of this test.
	Description string        `xml:"description"`   // Description of this test.
	CriticalTh  duration      `xml:"critical,attr"` // Duration above the critical level.
	WarningTh   duration      `xml:"warning,attr"`  // Duration above the warning level.
	Requests    []*RequestXML `xml:"request"`       // Top-level requests for this test.
	offspring   *Offspring    // Offspring of sent top-level requests.
}

// RequestXML stores the request as XML.
// It is kept in XML until it is executed to read from the parent response as needed.
type RequestXML struct {
	Parent      *RequestXML     // Parent of this request, can be nil.
	Children    []*RequestXML   `xml:"request"`               // Children of this request.
	Method      string          `xml:"method,attr"`           // Method of this request.
	Repeat      int             `xml:"repeat,attr"`           // Number of times to repeat this request.
	Concurrency int             `xml:"concurrency,attr"`      // Number of concurrent requests like these to send.
	RespType    string          `xml:"responseType,attr"`     // Response type which can be used for child requests.
	FwdCookies  bool            `xml:"useParentCookies,attr"` // Forward the parent response cookies to the children requests.
	URL         *URL            `xml:"url"`                   // URL to request.
	Headers     *Tokenized      `xml:"headers"`               // Headers to send.
	Data        *Tokenized      `xml:"data"`                  // Data to send.
	startTime   time.Time       // Start time of this request.
	duration    time.Duration   // Stores the duration of the fetch in nanoseconds.
	offspring   *Offspring      // Offspring of sent children requests
	resp        *goreq.Response // Response from the request.
}

// Validate confirms that a request is correctly defined.
func (r *RequestXML) Validate() {
	if r.Concurrency > r.Repeat {
		panic(fmt.Errorf("concurrency of %d for %d repetitions does not make sense", r.Concurrency, r.Repeat))
	}
	if r.Method == "" {
		panic("method not defined")
	}
	if r.RespType != "" && r.RespType != "json" {
		panic(fmt.Errorf("reponseType `%s` is not yet supported", r.RespType))
	}
	r.Method = strings.ToUpper(r.Method)
	r.URL.Validate()
}

// Spawn sends the actual request.
func (r *RequestXML) Spawn(parent *goreq.Response) {
	r.startTime = time.Now()
	req := goreq.Request{Method: r.Method, Uri: r.URL.Generate(), Body: r.Data.Format(parent)}
	// Let's set the headers.
	for _, line := range strings.Split(r.Headers.Format(parent), "\n") {
		hdr := strings.Split(line, ":")
		req.AddHeader(strings.TrimSpace(hdr[0]), strings.TrimSpace(hdr[1]))
	}
	// Let's also add the cookies.
	if r.FwdCookies {
		if parent.Cookies() != nil {
			for _, delicacy := range parent.Cookies() {
				req.AddCookie(delicacy)
			}
		}
	}
	resp, err := req.Do()
	if err != nil {
		log.Critical("could not send request to %s: %s", req.Uri, err)
		return
	}
	r.duration = time.Now().Sub(r.startTime)
	r.resp = resp
	r.offspring = &Offspring{}
	r.offspring.Breed(r) // Is this right?!
}

// URL handles URL generation based on the requested pattern.
type URL struct {
	Base   string      `xml:"base,attr"`
	Tokens *[]URLToken `xml:"token"`
}

// Validate confirms the validity of a URL.
func (u *URL) Validate() {
	if u.Tokens != nil {
		for _, tok := range *u.Tokens {
			tok.Validate()
			if !strings.Contains(u.Base, tok.Token) {
				panic(fmt.Errorf("cannot find token `%s` in base %s", tok.Token, u.Base))
			}
		}
	}
}

// Generate returns a new URL based on the base and the tokens.
func (u URL) Generate() (url string) {
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

// Tokenized stores the data handling from a given response.
type Tokenized struct {
	Response string `xml:"responseToken,attr"`
	Header   string `xml:"headerToken,attr"`
	Cookie   string `xml:"cookieToken,attr"`
	Data     string `xml:",innerxml"`
}

// Format returns the tokenized's data from a given response.
// Note: this does not use a pointer to not overwrite the initial Data.
func (t Tokenized) Format(resp *goreq.Response) (formatted string) {
	formatted = t.Data
	if t.Cookie != "" {
		// Setting the data from the cookies.
		cookies := map[string]string{}
		for _, cookie := range resp.Cookies() {
			cookies[cookie.Name] = cookie.Value
		}
		re := regexp.MustCompile(fmt.Sprintf("(?:%s/)([A-Za-z0-9-_]+)", t.Cookie))
		for _, match := range re.FindAllStringSubmatch(formatted, -1) {
			formatted = strings.Replace(formatted, match[0], cookies[match[1]], -1)
		}
	}
	if t.Header != "" {
		// Setting the data from the header.
		re := regexp.MustCompile(fmt.Sprintf("(?:%s/)([A-Za-z0-9-_]+)", t.Header))
		for _, match := range re.FindAllStringSubmatch(formatted, -1) {
			formatted = strings.Replace(formatted, match[0], resp.Header.Get(match[1]), -1)
		}
	}
	if t.Response != "" {
		// Changing the values based on the response.
		// The following allow us to only decode the data we need as a string (and hoping it is).
		jsonResp := map[string]json.RawMessage{}
		resp.Body.FromJsonTo(&jsonResp)
		re := regexp.MustCompile(fmt.Sprintf("(?:%s/)([A-Za-z0-9-_]+)", t.Response))
		for _, match := range re.FindAllStringSubmatch(formatted, -1) {
			var value string
			err := json.Unmarshal(jsonResp[match[1]], &value)
			if err != nil {
				log.Warning(fmt.Sprintf("could not convert response JSON field `%s` to a string", match[1]))
			}
			formatted = strings.Replace(formatted, match[0], string(value), -1)
		}
	}
	return
}

// loadProfile loads a profile XML file.
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
		if test.Requests == nil {
			return nil, fmt.Errorf("error loading profile %s: there are no requests to send\n", profileFile)
		}

		for _, request := range test.Requests {
			request.Validate()
			if request.FwdCookies {
				log.Warning("cannot use parent cookies in top request")
			}
			setParentRequest(request, request.Children)
		}
	}
	return &profile, nil
}

// setParentRequest sets the parent request recursively for all children.
func setParentRequest(parent *RequestXML, children []*RequestXML) {
	if children != nil {
		for _, child := range children {
			child.Validate()
			child.Parent = parent
			setParentRequest(child, child.Children)
		}
	}
}
