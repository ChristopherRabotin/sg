package main

import (
	"encoding/xml"
	"fmt"
	"sort"
	"time"
)

// Duration allows automatic unmarshaling of a duration from XML.
type Duration struct {
	Duration time.Duration `xml:"value,attr"`
	State    string        `xml:"state,attr"`
}

// UnmarshalXMLAttr unmarshals a duration.
func (dur *Duration) UnmarshalXMLAttr(attr xml.Attr) (err error) {
	parsed, err := time.ParseDuration(attr.Value)
	if err != nil {
		return
	}
	*dur = Duration{Duration: parsed}
	return nil
}

func (dur *Duration) String() string {
	return dur.Duration.String()
}

// MarshalXMLAttr implements the xml.MarshalerAttr interface.
func (dur *Duration) MarshalXMLAttr(name xml.Name) (attr xml.Attr, err error) {
	attr.Name = name
	attr.Value = dur.String()
	return
}

// Percentages stores some Percentagess.
type Percentages struct {
	MeanValue Duration `xml:"mean,attr"`
	P1        Duration `xml:"shortest,attr"`
	P10       Duration `xml:"p10,attr"`
	P25       Duration `xml:"p25,attr"`
	P50       Duration `xml:"p50,attr"`
	P66       Duration `xml:"p66,attr"`
	P75       Duration `xml:"p75,attr"`
	P80       Duration `xml:"p80,attr"`
	P90       Duration `xml:"p90,attr"`
	P95       Duration `xml:"p95,attr"`
	P98       Duration `xml:"p98,attr"`
	P99       Duration `xml:"p99,attr"`
	P100      Duration `xml:"longest,attr"`
	vals      []Duration
	length    int
}

// NewPercentages returns a stuct which helps in serializing request results.
func NewPercentages(vals []time.Duration) *Percentages {
	dVals := make([]Duration, len(vals))
	for i, v := range vals {
		dVals[i] = Duration{Duration: v}
	}
	p := Percentages{vals: dVals, length: len(vals)}
	sort.Sort(&p)
	p.P1 = Duration{Duration: p.Percentage(1)}
	p.P10 = Duration{Duration: p.Percentage(10)}
	p.P25 = Duration{Duration: p.Percentage(25)}
	p.P50 = Duration{Duration: p.Percentage(50)}
	p.P66 = Duration{Duration: p.Percentage(66)}
	p.P75 = Duration{Duration: p.Percentage(75)}
	p.P80 = Duration{Duration: p.Percentage(80)}
	p.P90 = Duration{Duration: p.Percentage(90)}
	p.P95 = Duration{Duration: p.Percentage(95)}
	p.P98 = Duration{Duration: p.Percentage(98)}
	p.P99 = Duration{Duration: p.Percentage(99)}
	p.P100 = Duration{Duration: p.Percentage(100)}
	return &p
}

// Len is for the Sorter interface.
func (p *Percentages) Len() int {
	return p.length
}

// Less is for the Sorter interface.
func (p *Percentages) Less(i, j int) bool {
	return p.vals[i].Duration < p.vals[j].Duration
}

// Swap is for the Sorter interface.
func (p *Percentages) Swap(i, j int) {
	p.vals[i], p.vals[j] = p.vals[j], p.vals[i]
}

// Percentage returns the items in position X.
func (p *Percentages) Percentage(v int) time.Duration {
	if v < 0 || v > 100 {
		panic(fmt.Errorf("incorrect value requested %f", v))
	}
	if v == 0 {
		return p.vals[0].Duration
	}
	if v == 100 {
		return p.vals[p.length-1].Duration
	}
	return p.vals[int(float64(p.length)*float64(v)/100)].Duration
}

// Mean returns the mean of this set of values.
func (p *Percentages) Mean() time.Duration {
	if p.MeanValue.Duration == 0 && p.length > 0 {
		sum := int64(0)
		for i := 0; i < p.length; i++ {
			sum += p.vals[i].Duration.Nanoseconds()
		}
		p.MeanValue.Duration = time.Duration(sum / int64(p.length))
	}
	return p.MeanValue.Duration
}

// SetState sets the state of each percentile based on the input parameters.
func (p *Percentages) SetState(critical time.Duration, warning time.Duration) {
	for i := 0; i < p.length; i++ {
		if p.vals[i].Duration >= critical {
			p.vals[i].State = "critical"
		} else if p.vals[i].Duration >= warning {
			p.vals[i].State = "warning"
		} else {
			p.vals[i].State = "nominal"
		}
	}
}

//String implements the Stringer interface.
func (p *Percentages) String() string {
	return fmt.Sprintf("Shortest: %s Median: %s Q3: %s P95: %s Longest: %s", p.P1.String(), p.P50.String(), p.P75.String(), p.P95.String(), p.P100.String())
}
