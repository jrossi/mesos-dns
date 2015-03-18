package logging

import (
	"io/ioutil"
	"log"
	"os"
)

var (
	Info    *log.Logger
	Verbose *log.Logger
	Error   *log.Logger
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

var CurLog LogOut

// PrintCurLog prints out the current LogOut and then resets
func PrintCurLog() {
	Verbose.Printf("%+v\n", CurLog)
}

func init() {
	Info = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Verbose = log.New(ioutil.Discard, "VERBOSE: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func VerboseEnable() {
	Verbose = log.New(os.Stdout, "VERBOSE: ", log.Ldate|log.Ltime|log.Lshortfile)
}
