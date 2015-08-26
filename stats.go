package main

import (
	"fmt"
	"sort"
)

// Percentages stores some Percentagess.
type Percentages struct {
	MeanValue  float64 `xml:"mean,attr"`
	P1         float64 `xml:"shortest,attr"`
	P10        float64 `xml:"p10,attr"`
	P25        float64 `xml:"p25,attr"`
	P50        float64 `xml:"p50,attr"`
	P66        float64 `xml:"p66,attr"`
	P75        float64 `xml:"p75,attr"`
	P80        float64 `xml:"p80,attr"`
	P90        float64 `xml:"p90,attr"`
	P95        float64 `xml:"p95,attr"`
	P98        float64 `xml:"p98,attr"`
	P99        float64 `xml:"p99,attr"`
	P100       float64 `xml:"longest,attr"`
	sortedVals []float64
	length     int
}

// NewPercentages returns a stuct which helps in serializing request results.
func NewPercentages(vals []float64) *Percentages {
	sort.Float64s(vals)
	p := Percentages{sortedVals: vals, length: len(vals)}
	p.P1 = p.Percentage(1)
	p.P10 = p.Percentage(10)
	p.P25 = p.Percentage(25)
	p.P50 = p.Percentage(50)
	p.P66 = p.Percentage(66)
	p.P75 = p.Percentage(75)
	p.P80 = p.Percentage(80)
	p.P90 = p.Percentage(90)
	p.P95 = p.Percentage(95)
	p.P98 = p.Percentage(98)
	p.P99 = p.Percentage(99)
	p.P100 = p.Percentage(100)
	return &p
}

// Percentage returns the items in position X.
func (p Percentages) Percentage(v int) float64 {
	if v < 0 || v > 100 {
		panic(fmt.Errorf("incorrect value requested %f", v))
	}
	if v == 0 {
		return p.sortedVals[0]
	}
	if v == 100 {
		return p.sortedVals[p.length-1]
	}
	return p.sortedVals[int(float64(p.length)*float64(v)/100)]
}

// Mean returns the mean of this set of values.
func (p *Percentages) Mean() float64 {
	if p.MeanValue == 0 && p.length > 0 {
		sum := 0.0
		for i := 0; i < p.length; i++ {
			sum += p.sortedVals[i]
		}
		p.MeanValue = sum / float64(p.length)
	}
	return p.MeanValue
}

//String implements the Stringer interface.
func (p *Percentages) String() string {
	return fmt.Sprintf("Shortest: %2.3fs Median: %2.3fs Q3: %2.3fs P95: %2.3fs Longest: %2.3fs", p.P1, p.P50, p.P75, p.P95, p.P100)
}
