package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestRequests(t *testing.T) {
	Convey("A request validation, ", t, func() {
		Convey("should panic if there is more concurrency than repetition", func() {
			r := Request{Concurrency: 2, Repeat: 1}
			So(r.Validate, ShouldPanic)
		})
		Convey("should panic if there is no method", func() {
			r := Request{Concurrency: 1, Repeat: 1}
			So(r.Validate, ShouldPanic)
		})
		Convey("should panic if the response type is not supported", func() {
			r := Request{Concurrency: 1, Repeat: 1, Method: "Not checked", RespType: "unsupported"}
			So(r.Validate, ShouldPanic)
		})
	})
}
