package main

import (
	"encoding/xml"
	"fmt"
	"sort"
	"time"
)

// Duration allows automatic unmarshaling of a duration from XML.
type Duration struct {
	Duration time.Duration
	State    string
}

// SetState sets the state of each percentile based on the input parameters.
func (dur *Duration) SetState(critical time.Duration, warning time.Duration) {
	if dur.Duration >= critical {
		dur.State = "critical"
	} else if dur.Duration >= warning {
		dur.State = "warning"
	} else {
		dur.State = "nominal"
	}
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

// MarshalXML is a custom marshaller for Duration.
func (dur *Duration) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = []xml.Attr{xml.Attr{Name: xml.Name{Local: "duration"}, Value: dur.Duration.String()},
		xml.Attr{Name: xml.Name{Local: "state"}, Value: dur.State},
	}
	e.EncodeToken(start)
	e.EncodeToken(xml.EndElement{Name: start.Name})
	return nil
}

// Percentages stores some Percentagess.
type Percentages struct {
	MeanValue Duration `xml:"mean"`
	vals      []*Duration
	length    int
}

func (p *Percentages) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeToken(start)
	p.MeanValue.MarshalXML(e, xml.StartElement{Name: xml.Name{Local: "mean"}})
	var elName string
	for _, perc := range []int{0, 10, 25, 50, 66, 75, 80, 90, 95, 98, 99, 100} {
		if perc == 0 {
			elName = "shortest"
		} else if perc == 100 {
			elName = "longest"
		} else {
			elName = fmt.Sprintf("p%d", perc)
		}
		p.Percentage(perc).MarshalXML(e, xml.StartElement{Name: xml.Name{Local: elName}})
	}
	e.EncodeToken(xml.EndElement{Name: start.Name})
	return nil
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
func (p *Percentages) Percentage(v int) *Duration {
	if v < 0 || v > 100 {
		panic(fmt.Errorf("incorrect value requested %f", v))
	}
	if v == 0 {
		return p.vals[0]
	}
	if v == 100 {
		return p.vals[p.length-1]
	}
	return p.vals[int(float64(p.length)*float64(v)/100)]
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
		p.vals[i].SetState(critical, warning)
	}
	p.MeanValue.SetState(critical, warning)
}

//String implements the Stringer interface.
func (p *Percentages) String() string {
	return fmt.Sprintf("Shortest: %s Median: %s Q3: %s P95: %s Longest: %s", p.Percentage(1).String(), p.Percentage(50).String(), p.Percentage(75).String(), p.Percentage(95).String(), p.Percentage(100).String())
}

// NewPercentages returns a stuct which helps in serializing request results.
func NewPercentages(vals []time.Duration) *Percentages {
	dVals := make([]*Duration, len(vals))
	for i, v := range vals {
		dVals[i] = &Duration{Duration: v}
	}
	p := Percentages{vals: dVals, length: len(vals)}
	sort.Sort(&p)
	return &p
}
