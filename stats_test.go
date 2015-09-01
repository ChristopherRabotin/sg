package main

import (
	"encoding/xml"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestPercentages(t *testing.T) {
	Convey("Testing percentages", t, func() {
		Convey("Given a slice of floats, the percentages should be correct", func() {
			myslice := make([]time.Duration, 100)
			for i := 0; i < 100; i++ {
				myslice[99-i] = time.Microsecond * time.Duration(i)
			}
			p := NewPercentages(myslice)
			for i := 0; i < 100; i++ {
				So(p.Percentage(i), ShouldEqual, time.Microsecond*time.Duration(i))
			}
			So(p.Mean(), ShouldEqual, time.Microsecond*time.Duration(49)+time.Nanosecond*500)
			b, _ := xml.Marshal(p)
			So(p.String(), ShouldEqual, `Shortest: 1µs Median: 50µs Q3: 75µs P95: 95µs Longest: 99µs`)
			So(string(b), ShouldEqual, `<Percentages mean="49.5µs" shortest="1µs" p10="10µs" p25="25µs" p50="50µs" p66="66µs" p75="75µs" p80="80µs" p90="90µs" p95="95µs" p98="98µs" p99="99µs" longest="99µs"></Percentages>`)
			So(func() { p.Percentage(-1) }, ShouldPanic)
		})
	})
}
