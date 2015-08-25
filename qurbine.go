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
	Parent      *Request        // Parent of this request, can be nil.
	Children    []*Request      `xml:"request"`               // Children of this request.
	Method      string          `xml:"method,attr"`           // Method of this request.
	Repeat      int             `xml:"repeat,attr"`           // Number of times to repeat this request.
	Concurrency int             `xml:"concurrency,attr"`      // Number of concurrent requests like these to send.
	RespType    string          `xml:"responseType,attr"`     // Response type which can be used for child requests.
	FwdCookies  bool            `xml:"useParentCookies,attr"` // Forward the parent response cookies to the children requests.
	URL         *URL            `xml:"url"`                   // URL to request.
	Headers     *Tokenized      `xml:"headers"`               // Headers to send.
	Data        *Tokenized      `xml:"data"`                  // Data to send.
	Result      *Result         `xml:"result"`
	startTime   time.Time       // Start time of this request.
	duration    time.Duration   // Stores the duration of the fetch in nanoseconds.
	offspring   *Offspring      // Offspring of sent children requests
	resp        *goreq.Response // Response from the request.
}

// Validate confirms that a request is correctly defined.
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
}

// Spawn sends the actual request.
func (r *Request) Spawn(parent *goreq.Response, wg *sync.WaitGroup) {
	body := ""
	if r.Data != nil {
		body = r.Data.Format(parent)
	}
	req := goreq.Request{Method: r.Method, Uri: r.URL.Generate(), Body: body}
	// Let's set the headers, if needed.
	if r.Headers != nil {
		for _, line := range strings.Split(r.Headers.Format(parent), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			hdr := strings.Split(line, ":")
			req.AddHeader(strings.TrimSpace(hdr[0]), strings.TrimSpace(hdr[1]))
		}
	}
	// Let's also add the cookies.
	if r.FwdCookies && parent != nil {
		if parent.Cookies() != nil {
			for _, delicacy := range parent.Cookies() {
				req.AddCookie(delicacy)
			}
		}
	}

	totalSentRequests++
	r.startTime = time.Now()
	resp, err := req.Do()
	r.duration = time.Now().Sub(r.startTime)
	log.Info("%s lasted %s", r.URL, r.duration)
	if err != nil {
		log.Critical("could not send request to %s: %s", req.Uri, err)
		return
	}
	r.resp = resp
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

// Offspring handles bred requests.
type Offspring struct {
	Ongoing   map[*Request]chan struct{} // Stores the channel of a given request.
	Completed map[*Request][]*Request    // Stores the channel of a given request.
}

// NewOffspring initializes an offspring.
func NewOffspring() (o *Offspring) {
	o = &Offspring{}
	o.Ongoing = make(map[*Request]chan struct{})
	o.Completed = make(map[*Request][]*Request)
	return
}

// Breed generates child requests.
func (o *Offspring) Breed(r *Request, wg *sync.WaitGroup) {
	o.Ongoing[r] = make(chan struct{}, r.Concurrency)
	o.Completed[r] = make([]*Request, 0)
	for rno := 0; rno < r.Repeat; rno++ {
		wg.Add(1)
		// TODO: Fix logic in the go routine.
		// Issue 1: after this is executed, everything in o.Completed is actually the same.
		// This is not surprising once this was re-written using the go routine parameter (instead of a mix of r and req).
		// Shouldn't the parent request be used somewhere here?
		// Issue 2: the output shows that there is only one result per parent request instead of a hierarchy.
		// Issue 3: The output XML is far too verbose and should really only contain the results.
		go func(req *Request) {
			o.Ongoing[req] <- struct{}{} // Adding sentinel value to limit concurrency.
			req.Spawn(req.resp, wg)
			<-o.Ongoing[req]
			o.Completed[req] = append(o.Completed[req], req)
			perc := float64(len(o.Completed[req])) / float64(req.Repeat)
			notify := false
			if perc >= 0.75 && perc-0.75 < 0.01 {
				notify = true
			} else if perc >= 0.5 && perc-0.5 < 0.01 {
				notify = true
			} else if perc >= 0.25 && perc-0.25 < 0.01 {
				notify = true
			} else if len(o.Completed[req])%100 == 0 {
				notify = true
			}
			if notify {
				log.Debug("Completed %d requests out of %d to %s.", len(o.Completed[req]), req.Repeat, req.URL)
			}
			if len(o.Completed[req]) == req.Repeat {
				// Let's breed the children, if applicable.
				// WARNING: the children will use the LAST response for their requests.
				r.offspring = NewOffspring()
				if r.Children != nil {
					for _, child := range r.Children {
						req.offspring.Breed(child, wg)
					}
				}
				// All the requests have completed, let's compute the results for this request.
				o.ComputeResult(req, wg)
			}
		}(r)
	}
}

// ComputeResult computes the results of the given request.
func (o *Offspring) ComputeResult(r *Request, wg *sync.WaitGroup) {
	times := []float64{}
	statuses := make(map[int]Status)
	summary := StatusSummary{}
	for i, child := range o.Completed[r] {
		times = append(times, child.duration.Seconds())
		log.Info("child %d %s", i, child.duration)
		if child.resp == nil {
			// An error occurred when executing this request.
			summary.None++
		} else {
			if val, exists := statuses[child.resp.StatusCode]; exists {
				val.Count++
				statuses[child.resp.StatusCode] = val
			} else {
				statuses[child.resp.StatusCode] = Status{Code: child.resp.StatusCode, Count: 1}
			}
			switch child.resp.StatusCode / 100 {
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
				log.Warning("Unsupported status code %d received.", child.resp.StatusCode)
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
