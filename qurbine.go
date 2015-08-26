package main

import (
	"fmt"
	"github.com/franela/goreq"
	"strings"
	"sync"
	"time"
)

// Request stores the request as XML.
// It is kept in XML until it is executed to read from the parent response as needed.
type Request struct {
	Parent       *Request       // Parent of this request, can be nil.
	Children     []*Request     `xml:"request"`               // Children of this request.
	Method       string         `xml:"method,attr"`           // Method of this request.
	Repeat       int            `xml:"repeat,attr"`           // Number of times to repeat this request.
	Concurrency  int            `xml:"concurrency,attr"`      // Number of concurrent requests like these to send.
	RespType     string         `xml:"responseType,attr"`     // Response type which can be used for child requests.
	FwdCookies   bool           `xml:"useParentCookies,attr"` // Forward the parent response cookies to the children requests.
	URL          *URL           `xml:"url"`                   // URL to request.
	Headers      *Tokenized     `xml:"headers"`               // Headers to send.
	Data         *Tokenized     `xml:"data"`                  // Data to send.
	Result       *Result        `xml:"result"`
	duration     time.Duration  // Stores the duration of the fetch in nanoseconds.
	ongoingReqs  chan struct{}  // Channel of ongoing requests.
	doneChan     chan *Response // Channel of responses to buffer them prior to transfering them to doneReqs.
	doneReqs     []*Response    // List of responses.
	numCompleted int            // Number of completed requests.
	doneWg       sync.WaitGroup // Wait group of the completed requests.
}

// Validate confirms that a request is correctly defined and initializes variables.
func (r *Request) Validate() {
	if r.Concurrency > r.Repeat {
		panic(fmt.Errorf("concurrency of %d for %d repetitions does not make sense", r.Concurrency, r.Repeat))
	}
	if r.Method == "" {
		panic("method not defined")
	}
	if r.RespType != "" && r.RespType != "json" {
		panic(fmt.Errorf("reponseType `%s` is not yet supported", r.RespType))
	}
	r.Method = strings.ToUpper(r.Method)
	r.URL.Validate()
	r.ongoingReqs = make(chan struct{}, r.Concurrency)
	r.doneReqs = make([]*Response, 0)
	r.doneChan = make(chan *Response, r.Repeat)
}

// Spawn sends the actual request.
func (r *Request) Spawn(parent *goreq.Response, wg *sync.WaitGroup) {
	body := ""
	if r.Data != nil {
		body = r.Data.Format(parent)
	}
	greq := goreq.Request{Method: r.Method, Uri: r.URL.Generate(), Body: body}
	// Let's set the headers, if needed.
	if r.Headers != nil {
		for _, line := range strings.Split(r.Headers.Format(parent), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			hdr := strings.Split(line, ":")
			greq.AddHeader(strings.TrimSpace(hdr[0]), strings.TrimSpace(hdr[1]))
		}
	}
	// Let's also add the cookies.
	if r.FwdCookies && parent != nil {
		if parent.Cookies() != nil {
			for _, delicacy := range parent.Cookies() {
				greq.AddCookie(delicacy)
			}
		}
	}

	// One go routine which pops stuff from the channel and moves them to the list.
	go func() {
		for {
			r.doneReqs = append(r.doneReqs, <-r.doneChan)
			r.doneWg.Done()
			r.numCompleted++
			perc := float64(r.numCompleted) / float64(r.Repeat)
			notify := false
			if perc >= 0.75 && perc-0.75 < 0.01 {
				notify = true
			} else if perc >= 0.5 && perc-0.5 < 0.01 {
				notify = true
			} else if perc >= 0.25 && perc-0.25 < 0.01 {
				notify = true
			} else if len(r.doneReqs)%100 == 0 {
				notify = true
			}
			if notify {
				log.Notice("Completed %d requests out of %d to %s.", len(r.doneReqs), r.Repeat, r.URL)
			}
		}
	}()

	// Let's spawn all the requests, with their respective concurrency.
	for rno := 1; rno <= r.Repeat; rno++ {
		wg.Add(1)
		r.doneWg.Add(1)
		go func(no int) {
			r.ongoingReqs <- struct{}{} // Adding sentinel value to limit concurrency
			startTime := time.Now()
			gresp, err := greq.Do()
			<-r.ongoingReqs // We're done, let's make room for the next request.
			resp := Response{Response: gresp, duration: time.Now().Sub(startTime)}
			log.Debug("Request #%d to %s lasted %s.", no, r.URL, resp.duration)
			// Let's add that request to the list of completed requests.
			r.doneChan <- &resp
			if err != nil {
				log.Critical("could not send request to %s: %s", greq.Uri, err)
			}
		}(rno)
	}

	// Let's now have a go routine which waits for all the requests to complete
	// and spawns all the children.
	go func() {
		r.doneWg.Wait()
		if r.Children != nil {
			log.Debug("Spawning children for %s.", r.URL)
			for _, child := range r.Children {
				// Note that we always use the LAST response as the parent response.
				child.Spawn(r.doneReqs[len(r.doneReqs)-1].Response, wg)
			}
		}
		log.Debug("Computing result of %s.", r.URL)
		r.ComputeResult(wg)
	}()
}

// ComputeResult computes the results for the given request.
func (r *Request) ComputeResult(wg *sync.WaitGroup) {
	times := []float64{}
	statuses := make(map[int]Status)
	summary := StatusSummary{}
	for _, response := range r.doneReqs {
		totalSentRequests++
		times = append(times, response.duration.Seconds())
		if response.Response == nil {
			// An error occurred when executing this request.
			summary.None++
		} else {
			if val, exists := statuses[response.Response.StatusCode]; exists {
				val.Count++
				statuses[response.Response.StatusCode] = val
			} else {
				statuses[response.Response.StatusCode] = Status{Code: response.Response.StatusCode, Count: 1}
			}
			switch response.Response.StatusCode / 100 {
			case 1:
				summary.S1xx++
			case 2:
				summary.S2xx++
			case 3:
				summary.S3xx++
			case 4:
				summary.S4xx++
			case 5:
				summary.S5xx++
			default:
				log.Warning("Unsupported status code %d received.", response.Response.StatusCode)
			}
		}
		wg.Done()
	}
	// Let's aggregate all this in a Result object.
	result := Result{Method: r.Method, URL: r.URL.String(), Concurrency: r.Concurrency, Repetitions: r.Repeat,
		HadCookies: r.FwdCookies,
		HadData:    r.Data != nil && r.Data.IsUsed(),
		HadHeader:  r.Headers != nil && r.Headers.IsUsed(),
		StatusSum:  &summary,
		Times:      NewPercentages(times)}

	log.Notice("SUMMARY: %s %s", r, result.Times)

	statusesVals := make([]Status, len(statuses))
	i := 0
	for _, s := range statusesVals {
		statusesVals[i] = s
		i++
	}
	result.Statuses = statusesVals
	r.Result = &result
	// Let's now unset the children because we don't need them anymore.
	r.Children = nil
}

// String implements the Stringer interface.
func (r *Request) String() string {
	return fmt.Sprintf("%d request(s) (concurrency=%d) to %s", r.Repeat, r.Concurrency, r.URL)
}

// setParentRequest sets the parent request recursively for all children.
func setParentRequest(parent *Request, children []*Request) {
	if children != nil {
		for _, child := range children {
			child.Validate()
			child.Parent = parent
			setParentRequest(child, child.Children)
		}
	}
}

// Response extends a goreq.Response with a duration.
type Response struct {
	*goreq.Response
	duration time.Duration
}

// Result store the result of a group of requests (as define by its concurrency and repetition).
type Result struct {
	Method      string         `xml:"method,attr"`
	URL         string         `xml:"url,attr"`
	Concurrency int            `xml:"concurrency,attr"`
	Repetitions int            `xml:"repetitions,attr"`
	Times       *Percentages   `xml:"times"`
	Statuses    []Status       `xml:"status"`
	StatusSum   *StatusSummary `xml:"statuses"`
	HadCookies  bool           `xml:"withCookies,attr"`
	HadHeader   bool           `xml:"withHeaders,attr"`
	HadData     bool           `xml:"withData,attr"`
}

// Status stores the number of times a given status was found.
type Status struct {
	Code  int `xml:"code,attr"`
	Count int `xml:"number,attr"`
}

// StatusSummary stores the summary of statuses got for a group of requests.
type StatusSummary struct {
	None int `xml:"errored,attr"`
	S1xx int `xml:"s1xx,attr"`
	S2xx int `xml:"s2xx,attr"`
	S3xx int `xml:"s3xx,attr"`
	S4xx int `xml:"s4xx,attr"`
	S5xx int `xml:"s5xx,attr"`
}
