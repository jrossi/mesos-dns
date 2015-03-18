package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mesosphere/mesos-dns/logging"
	"github.com/mesosphere/mesos-dns/records"
	"github.com/mesosphere/mesos-dns/resolver"

	"github.com/codegangsta/cli"
	"github.com/miekg/dns"
)

func main() {
	app := cli.NewApp()
	app.Name = "mesos-dns"
	// From version.go can also be set via -ldflags
	app.Version = version
	app.Usage = "DNS-based service discovery for Mesos"
	app.Flags = records.Flags("MESOS_DNS")
	app.Action = func(c *cli.Context) {

		conf := records.New()

		// If config file load the json
		if c.String("jsonconfig") != "" {
			logging.Info.Println("Loading config: ", c.String("jsonconfig"))
			if err := conf.LoadFromFile(c.String("jsonconfig")); err != nil {
				logging.Error.Println("Loading File", c.String("jsonconfig"), "error", err)
				os.Exit(1)
			}
		}
		// Command line args and enironmnet override defaults, and jsonconfig
		if err := conf.LoadFromContext(c); err != nil {
			fmt.Println("Error Loading: ", err)
			os.Exit(1)
		}
		// Verify the config
		if err := conf.Check(); err != nil {
			fmt.Println("Error in Config: ", err)
			os.Exit(1)
		}

		if conf.Verbose {
			logging.VerboseEnable()
		}
		logging.Verbose.Println("Mesos-DNS configuration:")
		logging.Verbose.Println("   - Masters: " + strings.Join(conf.Masters, ", "))
		logging.Verbose.Println("   - RefreshSeconds: ", conf.RefreshSeconds)
		logging.Verbose.Println("   - TTL: ", conf.TTL)
		logging.Verbose.Println("   - Domain: " + conf.Domain)
		logging.Verbose.Println("   - Port: ", conf.Port)
		logging.Verbose.Println("   - Timeout: ", conf.Timeout)
		logging.Verbose.Println("   - Listener: " + conf.Listener)
		logging.Verbose.Println("   - Resolvers: " + strings.Join(conf.Resolvers, ", "))
		logging.Verbose.Println("   - Email: " + conf.Email)
		logging.Verbose.Println("   - Mname: " + conf.Mname)

		Server(conf)

	}
	app.Run(os.Args)

}

func Server(c records.Config) {
	var wg sync.WaitGroup
	var resolver resolver.Resolver

	resolver.Config = c

	// reload the first time
	resolver.Reload()
	ticker := time.NewTicker(time.Second * time.Duration(resolver.Config.RefreshSeconds))
	go func() {
		for _ = range ticker.C {
			resolver.Reload()
			logging.PrintCurLog()
		}
	}()

	// handle for everything in this domain...
	dns.HandleFunc(resolver.Config.Domain+".", panicRecover(resolver.HandleMesos))
	dns.HandleFunc(".", panicRecover(resolver.HandleNonMesos))

	go resolver.Serve("tcp")
	go resolver.Serve("udp")

	wg.Add(1)
	wg.Wait()
}

// panicRecover catches any panics from the resolvers and sets an error
// code of server failure
func panicRecover(f func(w dns.ResponseWriter, r *dns.Msg)) func(w dns.ResponseWriter, r *dns.Msg) {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		defer func() {
			if rec := recover(); rec != nil {
				m := new(dns.Msg)
				m.SetReply(r)
				m.SetRcode(r, 2)
				_ = w.WriteMsg(m)
				logging.Error.Println(rec)
			}
		}()
		f(w, r)
	}
}
