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

var totalSentRequests int

// init parses the flags.
func init() {
	totalSentRequests = 0
	flag.StringVar(&profileFile, "profile", "", "path to stress profile")
	logFormat := logging.MustStringFormatter("%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level}%{color:reset} %{message}")
	logging.SetBackend(logging.NewBackendFormatter(logging.NewLogBackend(os.Stderr, "", 0), logFormat))
}

func main() {
	flag.Parse()
	profile, err := loadProfile(profileFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for _, test := range profile.Tests {
		log.Notice("Starting test %s.", test)
		test.offspring = NewOffspring()
		for _, r := range test.Requests {
			test.offspring.Breed(r, &completionWg)
		}
	}
	completionWg.Wait()

	log.Info("Sent a total of %d requests.", totalSentRequests)
	saveResult(profile, profileFile)
}
