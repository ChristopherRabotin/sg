package sg

import ()

// Offspring handles stuff.
type Offspring struct {
	Ongoing    map[*RequestXML]chan struct{}    // Stores the channel of a given request.
	Completed  map[*RequestXML]chan *RequestXML // Stores the channel of a given request.
	ServedTime map[*RequestXML]*Percentages     `xml:"percentages"`
	Mean       int64                            `xml:"mean"`
}

// Must make the channel based on the concurrency for that request.
func (o *Offspring) Breed(r *RequestXML) {
	if r.Children == nil {
		// Nothing to breed.
		return
	}
	o.Ongoing[r] = make(chan struct{}, r.Concurrency)
	o.Completed[r] = make(chan *RequestXML, r.Repeat)
	for _, child := range r.Children {
		go func(child *RequestXML) {
			o.Ongoing[r] <- struct{}{}
			child.Spawn(r.resp)
			<-o.Ongoing[r]
			o.Completed[r] <- child
		}(child)
	}

	// Let's monitor the completed channels to check when we're done with everything.
	go func() {
		for {
			for r, c := range o.Completed {
				if len(c) == r.Repeat {
					// Let's compute all the values and close the channel.
					times := []float64{}
					for child := range c {
						times = append(times, child.duration.Seconds())
					}
					o.ServedTime[r] = NewPercentages(times)
					close(c)
				}
			}
		}
	}()
}
