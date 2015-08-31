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
		reqCount := 0
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqCount++
			requestHeaders = r.Header
			if r.Method == "GET" && r.URL.Path == "/init" {
				w.Header().Add("X-Custom-Hdr", "Custom Header")
				http.SetCookie(w, &http.Cookie{Value: "42", Name: "cookie_val"})
				w.WriteHeader(200)
				marsh, err := json.Marshal(GETTestJSON{URL: r.URL.String(), IsJson: true, BushJr: "shame on you"})
				if err != nil {
					panic(err)
				}
				fmt.Fprint(w, marsh)

			} else if r.Method == "POST" {
				if strings.HasPrefix("/cookie-fwd/", r.URL.String()) {
					cookie, err := r.Cookie("cookie_val")
					if err != nil {
						panic(err) // Causes the test to fail.
					} else if cookie.String() != "42" {
						panic(fmt.Errorf("invalid cookie value: %s", cookie.String()))
					} else {
						w.WriteHeader(200)
						marsh, err := json.Marshal(GETTestJSON{URL: r.URL.String()})
						if err != nil {
							panic(err)
						}
						fmt.Fprint(w, marsh)
					}
				} else if strings.HasPrefix("/header/", r.URL.String()) {
					if val := r.Header.Get("X-Custom-Hdr"); val == "" {
						panic(fmt.Errorf("invalid X-Custom-Hdr value: %s", val))
					} else {
						w.WriteHeader(200)
						marsh, err := json.Marshal(GETTestJSON{URL: r.URL.String(), CustomHdr: val})
						if err != nil {
							panic(err)
						}
						fmt.Fprint(w, marsh)
					}
				} else if strings.HasPrefix("/json/", r.URL.String()) {
					defer r.Body.Close()
					data, err := ioutil.ReadAll(r.Body)
					if err != nil {
						panic(err)
					}
					var marsh GETTestJSON
					json.Unmarshal(data, &marsh)
					if marsh.BushJr == "shame on you" {
						w.WriteHeader(204)
					} else {
						panic("no data received, or invalid data received")
					}
				} else if strings.HasPrefix("/slow/", r.URL.String()) {
					time.Sleep(time.Second * 2)
					w.WriteHeader(reqCount%6*100 + 4)
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
									<url base="json" />
									<data responseToken="resp" headerToken="hdr">
										{"foolMeOnce": "resp/foolMeOnce"}
									</data>
								</request>
							</request>
							<request method="get" responseType="json" repeat="100"
								concurrency="50">
								<url base="%s/slow/" />
							</request>
						</test>
					</sg>`, ts.URL, ts.URL, ts.URL, ts.URL)
		profile := Profile{}
		err := xml.Unmarshal([]byte(profileData), &profile)
		if err != nil {
			panic(err)
		}
		profile.Validate()
		stress(&profile)
	})
}
