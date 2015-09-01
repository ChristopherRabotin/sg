package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type GETTestJSON struct {
	URL       string `json:"URL"`
	IsJson    bool   `json:"json"`
	BushJr    string `json:"foolMeOnce"`
	CustomHdr string `json:"X-Custom-Hdr-Rcvd"`
}

func TestStressGauge(t *testing.T) {
	Convey("Stressing an HTTP Test server", t, func() {
		// Let's setup a test server.
		var ts *httptest.Server
		var requestHeaders http.Header
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestHeaders = r.Header
			if r.Method == "GET" {
				switch r.URL.Path {
				case "/init/":
					w.Header().Add("X-Custom-Hdr", "Custom Header")
					http.SetCookie(w, &http.Cookie{Value: "42", Name: "cookie_val"})
					marsh, err := json.Marshal(GETTestJSON{URL: r.URL.String(), IsJson: true, BushJr: "shame on you"})
					serveErrorOrBytes(w, err, marsh, t)
				case "/slow/":
					time.Sleep(time.Millisecond * 250)
					w.WriteHeader(((time.Now().Nanosecond()%6)+1)*100 + 4) // allows checking all the valid status codes
				}
			} else if r.Method == "POST" {
				switch r.URL.Path {
				case "/cookie-fwd/":
					cookie, err := r.Cookie("cookie_val")
					if isErrNil(w, err) {
						if cookie.Value != "42" {
							returnFailure("cookie value not 42", w, t)
						}
						marsh, err := json.Marshal(GETTestJSON{URL: r.URL.String()})
						serveErrorOrBytes(w, err, marsh, t)
					} else {
						returnFailure(fmt.Sprintf("cookie error: %s", err), w, t)
					}
				case "/header/":
					if val := r.Header.Get("X-Custom-Header"); val == "" {
						returnFailure(fmt.Sprintf("invalid X-Custom-Hdr value: %s", val), w, t)
					} else {
						marsh, err := json.Marshal(GETTestJSON{URL: r.URL.String(), CustomHdr: val})
						serveErrorOrBytes(w, err, marsh, t)
					}
				case "/json/":
					data, err := ioutil.ReadAll(r.Body)
					defer r.Body.Close()
					if isErrNil(w, err) {
						var marsh GETTestJSON
						err = json.Unmarshal(data, &marsh)
						if isErrNil(w, err) {
							if marsh.BushJr != "shame on you" {
								returnFailure(fmt.Sprintf("body is not `shame on you` but [%s] instead", marsh.BushJr), w, t)
							} else {
								w.WriteHeader(204)
							}
						} else {
							returnFailure(fmt.Sprintf("could not unmarshal JSON: %s", err), w, t)
						}
					} else {
						returnFailure("could not read body", w, t)
					}
				}
			}

		}))
		profileData := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
					<sg name="Basic example" uid="1">
						<test name="SG test" critical="1s" warning="750ms">
							<description>This is the test for SG.</description>
							<request method="get" responseType="json" repeat="1"
								concurrency="1">
								<url base="%s/init/" />
								<request method="post" responseType="json" repeat="20"
									useParentCookies="true" concurrency="10">
									<url base="%s/cookie-fwd/" />
								</request>
								<request method="post" responseType="json" repeat="20"
									useParentCookies="true" concurrency="10">
									<url base="%s/header/" />
									<headers headerToken="hdr">X-Custom-Header:hdr/X-Custom-Hdr
									</headers>
								</request>
								<request method="post" responseType="json" repeat="20"
									useParentCookies="true" concurrency="10">
									<url base="%s/json/" />
									<data responseToken="resp" headerToken="hdr">
										{"foolMeOnce": "resp/foolMeOnce"}
									</data>
								</request>
							</request>
							<request method="get" responseType="json" repeat="100"
								concurrency="50">
								<url base="%s/slow/" />
							</request>
							<request method="put" responseType="json" repeat="1"
								concurrency="1">
								<url base="%s-not-uri/error/" />
							</request>
						</test>
					</sg>`, ts.URL, ts.URL, ts.URL, ts.URL, ts.URL, ts.URL)
		profile := Profile{}
		err := xml.Unmarshal([]byte(profileData), &profile)
		if err != nil {
			panic(err)
		}
		profile.Validate()
		// Let's confirm that the children are set properly.
		checkNum := map[string]int{"init": 3, "slow": 0}
		for _, test := range profile.Tests {
			for _, req := range test.Requests {
				for url, count := range checkNum {
					if strings.Contains(req.URL.String(), url) {
						if len(req.Children) != count {
							panic("invalid number of children for init request")
						}
					}
				}
			}
		}
		stress(&profile)
	})
}

func isErrNil(w http.ResponseWriter, err error) bool {
	if err != nil {
		log.Critical("%s", err)
		w.WriteHeader(400)
		return false
	}
	return true
}

func serveErrorOrBytes(w http.ResponseWriter, err error, data []byte, t *testing.T) {
	if isErrNil(w, err) {
		w.Write(data)
	} else {
		returnFailure(fmt.Sprintf("%s", err), w, t)
	}
}

func returnFailure(msg string, w http.ResponseWriter, t *testing.T) {
	log.Error(msg)
	w.WriteHeader(400)
	t.Fail()
}
