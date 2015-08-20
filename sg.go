// Package sg is the main engine of the stress gauge.
package main

import (
	"flag"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("sg")

var profile string

func init() {
	flag.StringVar(&profile, "profile", "", "path to stress profile")
}

func main() {
	flag.Parse()
	if profile == "" {
		panic("profile flag not passed")
	}
}
