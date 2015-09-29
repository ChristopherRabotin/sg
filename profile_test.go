package main

import (
	"encoding/xml"
	"fmt"
	"github.com/franela/goreq"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestURL(t *testing.T) {
	Convey("The URL and URL Token tests, ", t, func() {

		Convey("No tokens should return the given base URL", func() {
			out := URL{}
			xml.Unmarshal([]byte(`<url base="http://example.org:7789/stress/PUT" />`), &out)
			So(out.Generate(), ShouldEqual, "http://example.org:7789/stress/PUT")
			So(out.String(), ShouldEqual, "http://example.org:7789/stress/PUT")
			So(out.Tokens, ShouldEqual, nil)
		})

		Convey("Invalid tokens", func() {
			Convey("Choices have no separator should fail validation", func() {
				u := URLToken{Choices: "Val1Val2", Token: "test"}
				So(u.Validate, ShouldPanic)
			})
			Convey("Choices have a min and max", func() {
				u := URLToken{Choices: "Val1|Val2", Token: "test", Min: 1}
				So(u.Validate, ShouldNotPanic)
				u = URLToken{Choices: "Val1|Val2", Token: "test", Max: 1}
				So(u.Validate, ShouldNotPanic)
				u = URLToken{Choices: "Val1|Val2", Token: "test", Min: 1, Max: 2}
				So(u.Validate, ShouldNotPanic)
			})
			Convey("Token is not defined", func() {
				u := URLToken{}
				So(u.Validate, ShouldPanic)
			})
			Convey("Choices and pattern not defined", func() {
				u := URLToken{Token: "test"}
				So(u.Validate, ShouldPanic)
			})
			Convey("Pattern does not exist", func() {
				u := URLToken{Pattern: "Val1Val2", Token: "test"}
				So(u.Validate, ShouldPanic)
			})
			Convey("Min is greater than max", func() {
				u := URLToken{Pattern: "alpha", Min: 10, Max: 5, Token: "test"}
				So(u.Validate, ShouldPanic)
			})
			Convey("Min and max are negative", func() {
				u := URLToken{Pattern: "alpha", Min: -10, Max: 5, Token: "test"}
				So(u.Validate, ShouldPanic)
				u = URLToken{Pattern: "alpha", Min: 10, Max: -5, Token: "test"}
				So(u.Validate, ShouldPanic)
				u = URLToken{Pattern: "alpha", Min: 10, Max: -5, Token: "test"}
				So(u.Validate, ShouldPanic)
			})
		})

		Convey("Valid tokens of all types should be deserialized correctly", func() {
			example := `<url base="http://example.org:1598/expensive/token1-token2/token3/token4">
					<token token="token1" choices="Val1|Val2" />
					<token token="token2" pattern="alpha" min="5" max="10" />
					<token token="token3" pattern="num" min="5" max="1000" />
					<token token="token4" pattern="alphanum" min="5" max="10" />
				</url>`
			out := URL{}
			xml.Unmarshal([]byte(example), &out)
			So(out.Tokens, ShouldNotEqual, nil)
			for _, tok := range *out.Tokens {
				So(tok.Validate, ShouldNotPanic)
				switch tok.Token {
				case "token1":
					So(tok.Choices, ShouldEqual, "Val1|Val2")
					So(tok.Pattern, ShouldEqual, "")
					So(tok.Min, ShouldEqual, 0)
					So(tok.Max, ShouldEqual, 0)
					So(tok.Generate(), ShouldBeIn, []string{"Val1", "Val2"})
				case "token2":
					So(tok.Choices, ShouldEqual, "")
					So(tok.Pattern, ShouldEqual, "alpha")
					So(tok.Min, ShouldEqual, 5)
					So(tok.Max, ShouldEqual, 10)
					matched, err := regexp.MatchString("[A-Za-z]{5,10}", tok.Generate())
					So(err, ShouldEqual, nil)
					So(matched, ShouldEqual, true)
					So(tok.Generate(), ShouldNotEqual, tok.Generate())
				case "token3":
					So(tok.Choices, ShouldEqual, "")
					So(tok.Pattern, ShouldEqual, "num")
					So(tok.Min, ShouldEqual, 5)
					So(tok.Max, ShouldEqual, 1000)
					matched, err := regexp.MatchString("[0-9]{1,4}", tok.Generate())
					So(err, ShouldEqual, nil)
					So(matched, ShouldEqual, true)
					So(tok.Generate(), ShouldNotEqual, tok.Generate())
					numTok, convErr := strconv.Atoi(tok.Generate())
					So(convErr, ShouldBeNil)
					So(numTok, ShouldBeGreaterThanOrEqualTo, 5)
					So(numTok, ShouldBeLessThan, 1000)
				case "token4":
					So(tok.Choices, ShouldEqual, "")
					So(tok.Pattern, ShouldEqual, "alphanum")
					So(tok.Min, ShouldEqual, 5)
					So(tok.Max, ShouldEqual, 10)
					matched, err := regexp.MatchString("[A-Za-z0-9]{5,10}", tok.Generate())
					So(err, ShouldEqual, nil)
					So(matched, ShouldEqual, true)
					So(tok.Generate(), ShouldNotEqual, tok.Generate())
				}
			}
			So(out.Validate, ShouldNotPanic)
			pattern := "http://example.org:1598/expensive/(Val1|Val2)-[A-Za-z]{5,10}/[0-9]{1,4}/[A-Za-z0-9]{5,10}"
			matched, err := regexp.MatchString(pattern, out.Generate())
			So(err, ShouldEqual, nil)
			So(matched, ShouldEqual, true)
			So(out.String(), ShouldEqual, pattern)
		})

		Convey("Non existing tokens should panic", func() {
			example := `<url base="http://example.org:1598/expensive/ToKeN1/">
					<token token="token1" choices="Val1|Val2" /></url>`
			out := URL{}
			xml.Unmarshal([]byte(example), &out)
			So(out.Tokens, ShouldNotEqual, nil)
			for _, tok := range *out.Tokens {
				// The token is valid so it should not panic.
				So(tok.Validate, ShouldNotPanic)
			}
			// However, if the token is not found in the URL, its validation should fail.
			So(out.Validate, ShouldPanic)

		})
	})
}

func TestLoadProfile(t *testing.T) {
	Convey("Loading profiles works as expected", t, func() {
		Convey("Loading nothing", func() {
			err := loadProfile("")
			So(err, ShouldNotEqual, nil)
		})
		Convey("Loading a non existing file", func() {
			err := loadProfile("this_file_does_not_exist")
			So(err, ShouldNotEqual, nil)
		})
		Convey("Loading the basic example", func() {
			err := loadProfile("./docs/examples/basic.xml")
			So(err, ShouldEqual, nil)
			// Let's now check that the basic example was loaded correctly.
			So(profile.Name, ShouldEqual, "Basic example")
			So(profile.UID, ShouldEqual, "1")
			numTests := 0
			for _, test := range profile.Tests {
				numTests++
				switch test.Name {
				case "Example 1":
					So(test.String(), ShouldEqual, "Example 1 (critical=1s, warning=750ms)")
					So(test.CriticalTh.Duration, ShouldEqual, time.Second*1)
					So(test.WarningTh.Duration, ShouldEqual, time.Millisecond*750)
					So(len(test.Requests), ShouldEqual, 1)
					So(test.Requests[0].Method, ShouldEqual, "POST")
					So(test.Requests[0].Repeat, ShouldEqual, 20)
					So(test.Requests[0].Concurrency, ShouldEqual, 10)
					So(test.Requests[0].RespType, ShouldEqual, "json")
					So(test.Requests[0].Headers, ShouldBeNil)
					So(test.Requests[0].Data, ShouldBeNil)
				case "Example 2":
					So(test.String(), ShouldEqual, "Example 2 (critical=1s, warning=750ms)")
					So(test.CriticalTh.Duration, ShouldEqual, time.Second*1)
					So(test.WarningTh.Duration, ShouldEqual, time.Millisecond*750)
					So(len(test.Requests), ShouldEqual, 1)
					So(test.Requests[0].Method, ShouldEqual, "POST")
					So(test.Requests[0].Repeat, ShouldEqual, 1)
					So(test.Requests[0].Concurrency, ShouldEqual, 1)
					So(test.Requests[0].RespType, ShouldEqual, "json")
					So(test.Requests[0].Headers.Data, ShouldEqual, "Cookie: example=true;")
					So(test.Requests[0].Data.Data, ShouldEqual, `{"username": "admin", "password": "superstrong"}`)
					So(test.Requests[0].FwdCookies, ShouldEqual, false)
					// Checking the subrequests.
					for pos, child := range test.Requests[0].Children {
						So(child.Parent, ShouldNotEqual, nil)
						So(child.Concurrency, ShouldEqual, 5)
						So(child.Repeat, ShouldEqual, 25)
						if pos == 0 {
							So(child.Method, ShouldEqual, "GET")
							So(child.FwdCookies, ShouldEqual, true)
						} else {
							So(child.Method, ShouldEqual, "PUT")
							So(child.FwdCookies, ShouldEqual, false)
						}
					}
				}
			}
		})
	})
}

func TestTokenized(t *testing.T) {
	Convey("Tokenizing data from a request works as expected", t, func() {
		// Let's setup a test server.
		var ts *httptest.Server
		var requestHeaders http.Header
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestHeaders = r.Header
			if r.Method == "GET" && r.URL.Path == "/test" {
				defer r.Body.Close()
				w.Header().Add("X-Custom-Hdr", "Custom Header")
				w.Header().Add("Set-Cookie", "session_id=42 ; Path=/")
				w.WriteHeader(200)
				fmt.Fprint(w, fmt.Sprintf(`{"URL": "%s", "json": true, "foolMeOnce": "shame on you"}`, r.URL))
			}
		}))
		Convey("Given a Tokenized objects, confirm that it formats the right information if does not format anything.", func() {
			t := Tokenized{Data: "test"}
			So(t.Format(nil), ShouldEqual, "test")
			So(t.String(), ShouldEqual, "{Tokenized with data}")
		})
		Convey("Given a Tokenized objects, confirm that it formats the right information if response is nil.", func() {
			t := Tokenized{Data: "test", Cookie: "ChocChip"}
			So(t.Format(nil), ShouldEqual, "test")
			So(t.String(), ShouldEqual, "{Tokenized with cookie with data}")
		})
		Convey("Given a Tokenized objects, confirm that it formats the right information.", func() {
			example := `<headers responseToken="resp" headerToken="hdr" cookieToken="cke">
							X-Fool:NotAMonkey resp/foolMeOnce
							Cookie:test=true;session_id=cke/session_id
							Some-Header:hdr/X-Custom-Hdr
							X-Cannot-Decode: resp/json
						</headers>`
			out := Tokenized{}
			xml.Unmarshal([]byte(example), &out)
			gresp, _ := goreq.Request{Uri: ts.URL + "/test"}.Do()
			resp := Response{}
			resp.FromGoResp(gresp, nil, time.Now())
			expectations := []string{"", "X-Fool:NotAMonkey shame on you", "Cookie:test=true;session_id=42", "Some-Header:Custom Header", "X-Cannot-Decode:", ""}
			for pos, line := range strings.Split(out.Format(&resp), "\n") {
				So(strings.TrimSpace(line), ShouldEqual, expectations[pos])
			}
			So(out.String(), ShouldEqual, "{Tokenized with cookie with header with data}")
		})
	})
}

func TestProfileConstraints(t *testing.T) {
	Convey("Profile validation should not be nominal", t, func() {
		Convey("there are no tests", func() {
			profileData := `<?xml version="1.0" encoding="UTF-8"?><sg name="Basic example" uid="1"><test name="Profile test" critical="1s" warning="750ms"/></sg>`
			profile := Profile{}
			xml.Unmarshal([]byte(profileData), &profile)
			So(profile.Validate(), ShouldNotBeNil)
		})
		Convey("there cookie forwaring is enabled on top request", func() {
			profileData := `<?xml version="1.0" encoding="UTF-8"?>
			<sg name="Basic example" uid="1">
				<test name="SG test" critical="1s" warning="750ms">
					<description>This is the test for SG.</description>
					<request method="get" responseType="json" repeat="20"
						useParentCookies="true" concurrency="10">
						<url base="http://google.com/search" />
					</request>
				</test>
			</sg>`
			profile := Profile{}
			xml.Unmarshal([]byte(profileData), &profile)
			So(func() { profile.Validate() }, ShouldNotPanic)
		})
	})
}
