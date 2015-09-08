package main

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

// Profile stores the whole test profile
type Profile struct {
	Name      string        `xml:"name,attr"`
	UID       string        `xml:"uid,attr"`
	UserAgent string        `xml:"user-agent,attr"`
	Tests     []*StressTest `xml:"test"`
}

// Validate confirms that a profile is valid and sets the parent to all children requests.
func (p *Profile) Validate() error {
	// Let's set the parent requests on all children.
	for _, test := range p.Tests {
		if test.Requests == nil || len(test.Requests) == 0 {
			return fmt.Errorf("error loading profile %s: there are no requests to send\n", profileFile)
		}

		for _, request := range test.Requests {
			request.Validate()
			if request.FwdCookies {
				log.Warning("using parent cookies in top request has no effect")
			}
			setParentRequest(request, request.Children)
		}
	}
	return nil
}

// StressTest stores the one stress test.
type StressTest struct {
	Name        string     `xml:"name,attr"`     // Name of this test.
	Description string     `xml:"description"`   // Description of this test.
	CriticalTh  Duration   `xml:"critical,attr"` // Duration above the critical level.
	WarningTh   Duration   `xml:"warning,attr"`  // Duration above the warning level.
	Requests    []*Request `xml:"request"`       // Top-level requests for this test.
	Result      *Result    `xml:"result"`        // Test results, populated only after the tests run.
}

func (t StressTest) String() string {
	return fmt.Sprintf("%s (critical=%s, warning=%s)", t.Name, t.CriticalTh.String(), t.WarningTh.String())
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

// String implements the Stringer interface.
func (u URL) String() (url string) {
	url = u.Base
	if u.Tokens != nil {
		for _, tok := range *u.Tokens {
			url = strings.Replace(url, tok.Token, tok.String(), -1)
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
func (t URLToken) Validate() {
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
func (t URLToken) Generate() (r string) {
	if t.Choices != "" {
		r, _ = randutil.ChoiceString(strings.Split(t.Choices, "|"))
	}
	switch t.Pattern {
	case "alpha":
		r, _ = randutil.StringRange(t.MinLength, t.MaxLength, randutil.Alphabet)
	case "alphanum":
		r, _ = randutil.AlphaStringRange(t.MinLength, t.MaxLength)
	case "num":
		rInt, _ := randutil.IntRange(int(math.Pow10(t.MinLength)), int(math.Pow10(t.MaxLength)))
		r = strconv.FormatInt(int64(rInt), 10)
	}
	return
}

func (t URLToken) String() string {
	if t.Choices != "" {
		return "(" + t.Choices + ")"
	}
	switch t.Pattern {
	case "alpha":
		return fmt.Sprintf("[A-Za-z]{%d,%d}", t.MinLength, t.MaxLength)
	case "alphanum":
		return fmt.Sprintf("[A-Za-z0-9]{%d,%d}", t.MinLength, t.MaxLength)
	case "num":
		return fmt.Sprintf("[0-9]{%d,%d}", t.MinLength, t.MaxLength)
	}
	return "" // can't happen
}

// Tokenized stores the data handling from a given response.
type Tokenized struct {
	Response string `xml:"responseToken,attr"`
	Header   string `xml:"headerToken,attr"`
	Cookie   string `xml:"cookieToken,attr"`
	Data     string `xml:",innerxml"`
}

// IsUsed returns whether this Tokenized will be computed.
func (t Tokenized) IsUsed() bool {
	return (t.Cookie != "" || t.Header != "" || t.Response != "")
}

// Format returns the tokenized's data from a given response.
// Note: this does not use a pointer to not overwrite the initial Data.
func (t Tokenized) Format(resp *goreq.Response) (formatted string) {
	formatted = t.Data
	if !t.IsUsed() {
		return
	}
	if resp == nil {
		log.Warning("Nothing to format for %s: response is nil.", t)
		return
	}
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
				log.Warning(fmt.Sprintf("could not convert response JSON field `%s` to a string: %s | json = %s", match[1], err, jsonResp))
			}
			formatted = strings.Replace(formatted, match[0], string(value), -1)
		}
	}
	return
}

func (t Tokenized) String() string {
	s := "{Tokenized"
	if t.Cookie != "" {
		s += " with cookie"
	}
	if t.Header != "" {
		s += " with header"
	}
	if t.Data != "" {
		s += " with data"
	}
	s += "}"
	return s
}

// loadProfile loads a profile XML file.
func loadProfile(profileFile string) error {
	if profileFile == "" {
		return errors.New("profile filename is empty")
	}
	profileData, err := ioutil.ReadFile(profileFile)
	if err != nil {
		return fmt.Errorf("error loading profile %s: %s\n", profileFile, err)
	}
	p := Profile{}
	if err = xml.Unmarshal(profileData, &p); err != nil {
		return fmt.Errorf("error loading profile %s: %s\n", profileFile, err)
	}

	profile = &p
	if err = p.Validate(); err != nil {
		return err
	}

	return nil
}

func saveResult(profile *Profile, profileFile string) string {
	// Let's move the top result from the request to the StressTest. TODO: Also set whether the duration is within constraints.
	content, err := xml.MarshalIndent(profile, "", "\t")
	if err != nil {
		log.Error("failed %+v", err)
		return ""
	}
	filename := fmt.Sprintf("%s-%s.xml", strings.Replace(profileFile, ".xml", "", -1), time.Now().Format("2006-01-02_1504"))
	ioutil.WriteFile(filename, content, 0644)
	return filename
}
