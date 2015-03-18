package records

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/mesosphere/mesos-dns/logging"

	"github.com/codegangsta/cli"
	"github.com/miekg/dns"
)

// Config holds mesos dns configuration
type Config struct {

	// Mesos master(s): a list of IP:port/zk pairs for one or more Mesos masters
	Masters []string `json:"masters"`

	// Refresh frequency: the frequency in seconds of regenerating records (default 60)
	RefreshSeconds int `json:"refreshseconds"`

	// TTL: the TTL value used for SRV and A records (default 60)
	TTL int `json:"ttl"`

	// Resolver port: port used to listen for slave requests (default 53)
	Port int `json:"port"`

	//  Domain: name of the domain used (default "mesos", ie .mesos domain)
	Domain string `json:"domain"`

	// DNS server: IP address of the DNS server for forwarded accesses
	Resolvers []string `json:"resolver"`

	// Timeout is the default connect/read/write timeout for outbound
	// queries
	Timeout int `json:"timeout"`

	// File is the location of the config.json file
	File string `json:"file"`

	// Email is the rname for a SOA
	Email string `json:"email"`

	// Mname is the mname for a SOA
	Mname string

	// ListenAddr is the server listener address
	Listener string

	// Verbose is the logging level
	Verbose     bool
	VeryVerbose bool
}

func Flags(prefix string) []cli.Flag {
	evname := func(n string) string {
		return fmt.Sprintf("%s_%s", strings.ToUpper(prefix), strings.ToUpper(n))
	}
	return []cli.Flag{
		cli.BoolFlag{
			Name:   "V, verbose",
			EnvVar: evname("VERBOSE"),
		},
		cli.BoolFlag{
			Name:   "VV, veryverbose",
			EnvVar: evname("VERYVERBOSE"),
		},
		cli.StringFlag{
			Name:   "j, jsonconfig",
			Usage:  "json configuration file",
			EnvVar: evname("JSONCONFIG"),
		},
		cli.StringFlag{
			Name:   "m, masters",
			Value:  "127.0.0.1:5050",
			Usage:  "comma sperated list of mesos master servers example: 1.1.1.1:5050,2.2.2.2:5050,3.3.3.3:5050",
			EnvVar: evname("MASTERS"),
		},
		cli.DurationFlag{
			Name:   "s, refreshseconds",
			Value:  time.Second,
			Usage:  "The frequency at which Mesos-DNS updates DNS records ",
			EnvVar: evname("REFRESH"),
		},
		cli.DurationFlag{
			Name:   "t, ttl",
			Value:  time.Second,
			Usage:  "The time to live value for DNS records served by Mesos-DNS",
			EnvVar: evname("TTL"),
		},
		cli.StringFlag{
			Name:   "d, domain",
			Value:  "mesos",
			Usage:  "The domain name for the Mesos cluster",
			EnvVar: evname("DOMAIN"),
		},
		cli.IntFlag{
			Name:   "p, port",
			Value:  53,
			Usage:  "The port number that Mesos-DNS monitors for incoming DNS requests",
			EnvVar: evname("PORT"),
		},
		cli.StringFlag{
			Name:   "r, resolver",
			Usage:  "A comma separated list with the IP addresses of external DNS servers that Mesos-DNS will contact to resolve any DNS requests outside the domain.",
			Value:  "",
			EnvVar: evname("RESOLVER"),
		},
		cli.DurationFlag{
			Name:   "T, timeout",
			Usage:  "The timeout threshold, requests to external DNS requests.",
			Value:  time.Second * 5,
			EnvVar: evname("TIMEOUT"),
		},
		cli.StringFlag{
			Name:   "l, listener",
			Usage:  "The IP address of Mesos-DNS. In SOA replies.",
			Value:  "0.0.0.0",
			EnvVar: evname("LISTENER"),
		},
		cli.StringFlag{
			Name:   "e, email",
			Usage:  "The email address of the Mesos domain name administrator. In Soa replies.",
			Value:  "root.mesos-dns.mesos",
			EnvVar: evname("EMAIL"),
		},
	}
}

func New() Config {
	return Config{
		RefreshSeconds: 60,
		TTL:            60,
		Domain:         "mesos",
		Port:           53,
		Timeout:        5,
		Email:          "root.mesos-dns.mesos",
		Resolvers:      []string{"8.8.8.8"},
		Listener:       "0.0.0.0",
	}
}

func (conf *Config) LoadFromContext(c *cli.Context) error {

	conf.Verbose = c.Bool("verbose")

	if c.IsSet("masters") {
		conf.Masters = strings.Split(c.String("masters"), ",")
	}

	if c.IsSet("refreshseconds") {
		conf.RefreshSeconds = int(c.Duration("refreshseconds").Seconds())
	}

	if c.IsSet("ttl") {
		conf.TTL = int(c.Duration("ttl").Seconds())
	}

	if c.IsSet("domain") {
		conf.Domain = c.String("domain")
	}

	if c.IsSet("port") {
		conf.Port = c.Int("port")
	}

	if c.IsSet("resolvers") {
		conf.Resolvers = strings.Split(c.String("resolvers"), ",")
	}

	if c.IsSet("timeout") {
		conf.Timeout = int(c.Duration("timeout").Seconds())
	}

	if c.IsSet("listener") {
		conf.Listener = c.String("listener")
	}

	if c.IsSet("email") {
		conf.Email = c.String("email")
	}

	return nil

}

func (conf *Config) LoadFromFile(cjson string) error {
	usr, _ := user.Current()
	dir := usr.HomeDir + "/"
	cjson = strings.Replace(cjson, "~/", dir, 1)

	path, err := filepath.Abs(cjson)
	if err != nil {
		logging.Error.Println("JSON File path error:", err)
		return err
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		logging.Error.Println("JSON file error:", err)
		return err
	}

	fmt.Println(string(b))

	err = json.Unmarshal(b, conf)
	if err != nil {
		logging.Error.Println("JSON Unmarshal Error:", err)
		return err
	}

	return nil
}

func (conf *Config) Check() error {

	if len(conf.Masters) == 0 {
		logging.Error.Println("please specify mesos masters in config.json, environment, or args")
		return errors.New("masters no in file, environment or args")
	}

	if conf.Port < 0 || conf.Port > 65535 {
		logging.Error.Println("Port must be between 0-65535")
		return errors.New("Port must be between 0-65535")
	}

	if len(conf.Resolvers) == 0 {
		conf.Resolvers = GetLocalDNS()
	}

	conf.Email = strings.Replace(conf.Email, "@", ".", -1)
	if conf.Email[len(conf.Email)-1:] != "." {
		conf.Email = conf.Email + "."
	}

	conf.Domain = strings.ToLower(conf.Domain)

	conf.Mname = "mesos-dns." + conf.Domain + "."

	return nil

}

// localAddies returns an array of local ipv4 addresses
func localAddies() []string {
	addies, err := net.InterfaceAddrs()
	if err != nil {
		logging.Error.Println(err)
	}

	bad := []string{}

	for i := 0; i < len(addies); i++ {
		ip, _, err := net.ParseCIDR(addies[i].String())
		if err != nil {
			logging.Error.Println(err)
		}
		t4 := ip.To4()
		if t4 != nil {
			bad = append(bad, t4.String())
		}
	}

	return bad
}

// nonLocalAddies only returns non-local ns entries
func nonLocalAddies(cservers []string) []string {
	bad := localAddies()

	good := []string{}

	for i := 0; i < len(cservers); i++ {
		local := false
		for x := 0; x < len(bad); x++ {
			if cservers[i] == bad[x] {
				local = true
			}
		}

		if !local {
			good = append(good, cservers[i])
		}
	}

	return good
}

// GetLocalDNS returns the first nameserver in /etc/resolv.conf
// used for out of mesos domain queries
func GetLocalDNS() []string {
	conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		logging.Error.Println(err)
		os.Exit(2)
	}

	return nonLocalAddies(conf.Servers)
}
