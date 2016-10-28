package main

import (
	"encoding/xml"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
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
				So(p.Percentage(i).Duration, ShouldEqual, time.Microsecond*time.Duration(i))
			}
			So(p.Mean(), ShouldEqual, time.Microsecond*time.Duration(49)+time.Nanosecond*500)
			So(p.String(), ShouldEqual, `Shortest: 1µs Median: 50µs Q3: 75µs P95: 95µs Longest: 99µs`)
			p.SetState(time.Nanosecond, time.Second)
			for i := 1; i < 100; i++ {
				So(p.vals[i].State, ShouldEqual, "critical")
			}
			p.SetState(time.Microsecond*time.Duration(75), time.Microsecond*time.Duration(50))
			for i := 1; i < 50; i++ {
				So(p.vals[i].State, ShouldEqual, "nominal")
			}
			for i := 50; i < 75; i++ {
				So(p.vals[i].State, ShouldEqual, "warning")
			}
			for i := 75; i < 100; i++ {
				So(p.vals[i].State, ShouldEqual, "critical")
			}
			b, _ := xml.Marshal(p)
			So(string(b), ShouldEqual, `<Percentages><mean duration="49.5µs" state="nominal"></mean><shortest duration="0s" state="nominal"></shortest><p10 duration="10µs" state="nominal"></p10><p25 duration="25µs" state="nominal"></p25><p50 duration="50µs" state="warning"></p50><p66 duration="66µs" state="warning"></p66><p75 duration="75µs" state="critical"></p75><p80 duration="80µs" state="critical"></p80><p90 duration="90µs" state="critical"></p90><p95 duration="95µs" state="critical"></p95><p98 duration="98µs" state="critical"></p98><p99 duration="99µs" state="critical"></p99><longest duration="99µs" state="critical"></longest></Percentages>`)
			So(func() { p.Percentage(-1) }, ShouldPanic)
		})
	})
}
