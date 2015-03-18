package logging

import (
	"io/ioutil"
	"log"
	"os"
)

var (
	Info        *log.Logger
	Verbose     *log.Logger
	VeryVerbose *log.Logger
	Error       *log.Logger
	logopts     int
	CurLog      LogOut
)

type LogOut struct {
	MesosRequests    int
	MesosSuccess     int
	MesosNXDomain    int
	MesosFailed      int
	NonMesosRequests int
	NonMesosSuccess  int
	NonMesosNXDomain int
	NonMesosFailed   int
	NonMesosRecursed int
}

// PrintCurLog prints out the current LogOut and then resets
func PrintCurLog() {
	VeryVerbose.Printf("%+v\n", CurLog)
}

func init() {
	logopts := log.Ldate | log.Ltime | log.Lshortfile

	Info = log.New(os.Stdout, "INFO: ", logopts)
	Verbose = log.New(ioutil.Discard, "VERBOSE: ", logopts)
	VeryVerbose = log.New(ioutil.Discard, "VERY VERBOSE: ", logopts)
	Error = log.New(os.Stderr, "ERROR: ", logopts)
}

func VerboseEnable() {
	Verbose = log.New(os.Stdout, "VERBOSE: ", logopts)
}

func VeryVerboseEnable() {
	VerboseEnable()
	VeryVerbose = log.New(os.Stdout, "VERY VERBOSE: ", logopts)
}
