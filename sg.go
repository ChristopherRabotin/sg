// Package main is the main engine of the stress gauge.
package main

import (
	"flag"
	"fmt"
	"github.com/op/go-logging"
	"os"
	"sync"
)

// log is the logger, duh.
var log = logging.MustGetLogger("sg")

// profileFile stores the filename of the profile to run.
var profileFile string

// completionWg is the completion wait group, which will wait for all requests to go through.
var completionWg sync.WaitGroup

// totalSentRequests stores the total number of sent requests.
var totalSentRequests int

// profile stores the profile to stress.
var profile *Profile

// init parses the flags.
func init() {
	totalSentRequests = 0
	flag.StringVar(&profileFile, "profile", "", "path to stress profile")
	logFormat := logging.MustStringFormatter("%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level}%{color:reset} %{message}")
	logging.SetBackend(logging.NewBackendFormatter(logging.NewLogBackend(os.Stderr, "", 0), logFormat))
}

func main() {
	flag.Parse()
	err := loadProfile(profileFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	stress(profile) // blocking call
	log.Notice("Saved output to %s.", saveResult(profile, profileFile))
}

func stress(profile *Profile) {
	for _, test := range profile.Tests {
		log.Notice("Starting test %s.", test)
		for _, r := range test.Requests {
			r.Spawn(nil, &completionWg)
		}
		completionWg.Wait()
	}

	log.Notice("Sent a total of %d requests.", totalSentRequests)
}
