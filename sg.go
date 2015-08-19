// Package sg is the main engine of the stress gauge.
package main

import (
	"flag"
)

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
