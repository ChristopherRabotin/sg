// Package sg is the main engine of the stress gauge.
package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/op/go-logging"
	"io/ioutil"
	"os"
)

var log = logging.MustGetLogger("sg")

var profileFile string

func init() {
	flag.StringVar(&profileFile, "profile", "", "path to stress profile")
}

func main() {
	flag.Parse()
	if profileFile == "" {
		fmt.Fprintln(os.Stderr, "profile flag not passed")
		return
	}
	profileData, err := ioutil.ReadFile(profileFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading profile %s: %s\n", profileFile, err)
		return
	}
	profile := Profile{}
	err = xml.Unmarshal(profileData, &profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading profile %s: %s\n", profileFile, err)
		return
	}
	fmt.Printf("%+v\n", profile)
}
