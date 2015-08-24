package sg

import (
	"encoding/xml"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestPercentages(t *testing.T) {
	Convey("Given a slice of floats, the percentages should be correct", t, func() {
		myslice := make([]float64, 100)
		for i := 0; i < 100; i++ {
			myslice[99-i] = float64(i)
		}
		p := NewPercentages(myslice)
		for i := 0; i < 100; i++ {
			So(p.Percentage(i), ShouldEqual, i)
		}
		So(p.Mean(), ShouldEqual, 49.5)
		b, _ := xml.Marshal(p)
		So(string(b), ShouldEqual, `<Percentages mean="49.5" p1="1" p10="10" p25="25" p50="50" p66="66" p75="75" p80="80" p90="90" p95="95" p98="98" p99="99" p100="99"></Percentages>`)
	})
}
