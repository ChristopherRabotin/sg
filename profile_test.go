package main

import (
	"encoding/xml"
	. "github.com/smartystreets/goconvey/convey"
	"regexp"
	"testing"
)

func TestURL(t *testing.T) {
	Convey("The URL and URL Token tests, ", t, func() {

		Convey("No tokens", func() {
			out := URL{}
			xml.Unmarshal([]byte(`<url base="http://example.org:7789/stress/PUT" />`), &out)
			So(out.Get(), ShouldEqual, "http://example.org:7789/stress/PUT")
			So(out.Tokens, ShouldEqual, nil)
		})

		Convey("Valid tokens of all type", func() {
			example := `<url base="http://example.org:1598/expensive/token1-token2/token3/token4">
					<token token="token1" choices="Val1|Val2" />
					<token token="token2" pattern="alpha" min="5" max="10" />
					<token token="token3" pattern="num" min="5" max="10" />
					<token token="token4" pattern="alphanum" min="5" max="10" />
				</url>`
			//matched, err := regexp.MatchString("http://example.org:1598/expensive/token1-[A-Za-z]{5,10}/token3/token4", "http://example.org:1598/expensive/token1-token/token3/token4")
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
		})
	})
}
