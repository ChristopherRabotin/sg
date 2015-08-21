// Package sg is the main engine of the stress gauge.
package sg

import (
	"flag"
	"fmt"
	"github.com/op/go-logging"
	"os"
)

var log = logging.MustGetLogger("sg")

var profileFile string

func init() {
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

	fmt.Printf("%+v\n", profile)
}
