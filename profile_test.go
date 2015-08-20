package main

import (
	"encoding/xml"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"regexp"
	"testing"
	"time"
)

func TestURL(t *testing.T) {
	Convey("The URL and URL Token tests, ", t, func() {

		Convey("No tokens should return the given base URL", func() {
			out := URL{}
			xml.Unmarshal([]byte(`<url base="http://example.org:7789/stress/PUT" />`), &out)
			So(out.Generate(), ShouldEqual, "http://example.org:7789/stress/PUT")
			So(out.Tokens, ShouldEqual, nil)
		})

		Convey("Invalid tokens", func() {
			Convey("Choices have no separator should fail validation", func() {
				u := URLToken{Choices: "Val1Val2", Token: "test"}
				So(u.Validate, ShouldPanic)
			})
			Convey("Choices have a min and max", func() {
				u := URLToken{Choices: "Val1|Val2", Token: "test", MinLength: 1}
				So(u.Validate, ShouldNotPanic)
				u = URLToken{Choices: "Val1|Val2", Token: "test", MaxLength: 1}
				So(u.Validate, ShouldNotPanic)
				u = URLToken{Choices: "Val1|Val2", Token: "test", MinLength: 1, MaxLength: 2}
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
				u := URLToken{Pattern: "alpha", MinLength: 10, MaxLength: 5, Token: "test"}
				So(u.Validate, ShouldPanic)
			})
			Convey("Min and max are negative", func() {
				u := URLToken{Pattern: "alpha", MinLength: -10, MaxLength: 5, Token: "test"}
				So(u.Validate, ShouldPanic)
				u = URLToken{Pattern: "alpha", MinLength: 10, MaxLength: -5, Token: "test"}
				So(u.Validate, ShouldPanic)
				u = URLToken{Pattern: "alpha", MinLength: 10, MaxLength: -5, Token: "test"}
				So(u.Validate, ShouldPanic)
			})
		})

		Convey("Valid tokens of all types should be deserialized correctly", func() {
			example := `<url base="http://example.org:1598/expensive/token1-token2/token3/token4">
					<token token="token1" choices="Val1|Val2" />
					<token token="token2" pattern="alpha" min="5" max="10" />
					<token token="token3" pattern="num" min="5" max="10" />
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
					So(tok.MinLength, ShouldEqual, 0)
					So(tok.MaxLength, ShouldEqual, 0)
					So(tok.Generate(), ShouldBeIn, []string{"Val1", "Val2"})
				case "token2":
					So(tok.Choices, ShouldEqual, "")
					So(tok.Pattern, ShouldEqual, "alpha")
					So(tok.MinLength, ShouldEqual, 5)
					So(tok.MaxLength, ShouldEqual, 10)
					matched, err := regexp.MatchString("[A-Za-z]{5,10}", tok.Generate())
					So(err, ShouldEqual, nil)
					So(matched, ShouldEqual, true)
				case "token3":
					So(tok.Choices, ShouldEqual, "")
					So(tok.Pattern, ShouldEqual, "num")
					So(tok.MinLength, ShouldEqual, 5)
					So(tok.MaxLength, ShouldEqual, 10)
					matched, err := regexp.MatchString("[0-9]{5,10}", tok.Generate())
					So(err, ShouldEqual, nil)
					So(matched, ShouldEqual, true)
				case "token4":
					So(tok.Choices, ShouldEqual, "")
					So(tok.Pattern, ShouldEqual, "alphanum")
					So(tok.MinLength, ShouldEqual, 5)
					So(tok.MaxLength, ShouldEqual, 10)
					matched, err := regexp.MatchString("[A-Za-z0-9]{5,10}", tok.Generate())
					So(err, ShouldEqual, nil)
					So(matched, ShouldEqual, true)
				}
			}
			So(out.Validate, ShouldNotPanic)
			matched, err := regexp.MatchString("http://example.org:1598/expensive/(Val1|Val2)-[A-Za-z]{5,10}/[0-9]{5,10}/[A-Za-z0-9]{5,10}", out.Generate())
			So(err, ShouldEqual, nil)
			So(matched, ShouldEqual, true)
		})
	})
}

func TestLoadProfile(t *testing.T) {
	Convey("Loading profiles works as expected", t, func() {
		Convey("Loading nothing", func() {
			_, err := loadProfile("")
			So(err, ShouldNotEqual, nil)
		})
		Convey("Loading a non existing file", func() {
			_, err := loadProfile("this_file_does_not_exist")
			So(err, ShouldNotEqual, nil)
		})
		Convey("Loading the basic example", func() {
			profile, err := loadProfile("./docs/examples/basic.xml")
			So(err, ShouldEqual, nil)
			// Let's now check that the basic example was loaded correctly.
			So(profile.Name, ShouldEqual, "Basic example")
			So(profile.UID, ShouldEqual, "1")
			numTests := 0
			for _, test := range profile.Tests {
				numTests++
				fmt.Printf("\n%+v\n", test)
				switch test.Name {
				case "Example 1":
					So(test.CriticalTh.Duration, ShouldEqual, time.Second*1)
				}
			}
		})
	})
}
