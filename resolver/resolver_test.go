package resolver

import (
	"encoding/json"
	"github.com/mesosphere/mesos-dns/logging"
	"github.com/mesosphere/mesos-dns/records"
	"github.com/miekg/dns"
	"io/ioutil"
	"strconv"
	"testing"
	"time"
)

func init() {
	logging.VerboseEnable()
}

// dig @127.0.0.1 -p 8053 "bob.*.mesos" ANY
func TestCleanWild(t *testing.T) {
	dom := "bob.*.mesos"

	stripped := cleanWild(dom)

	if stripped != "bob.mesos" {
		t.Error("not stripping domain")
	}
}

func TestSplitDomain(t *testing.T) {
	var res Resolver

	host, port := res.splitDomain("bob.com:12345")

	if host != "bob.com" {
		t.Error("not grabbing host")
	}

	if port != 12345 {
		t.Error("not grabbing port")
	}

}

func TestShuffleAnswers(t *testing.T) {
	var res Resolver

	m := new(dns.Msg)

	for i := 0; i < 10; i++ {
		name := "10.0.0." + strconv.Itoa(i) + ":1234"

		rr, err := res.formatA("blah.com", name)
		if err != nil {
			t.Error(err)
		}
		m.Answer = append(m.Answer, rr)
	}

	n := new(dns.Msg)
	c := make([]dns.RR, len(m.Answer))
	copy(c, m.Answer)
	n.Answer = c

	_ = shuffleAnswers(m.Answer)

	sflag := false
	// 10! chance of failing here
	for i := 0; i < 10; i++ {
		if n.Answer[i] != m.Answer[i] {
			sflag = true
			break
		}
	}

	if !sflag {
		t.Error("not shuffling")
	}
}

func fakeDNS(port int) (Resolver, error) {
	var res Resolver
	res.Config = records.Config{
		TTL:       60,
		Port:      port,
		Domain:    "mesos",
		Resolvers: records.GetLocalDNS(),
		Listener:  "127.0.0.1",
		Email:     "root.mesos-dns.mesos.",
		Mname:     "mesos-dns.mesos.",
	}

	b, err := ioutil.ReadFile("../factories/fake.json")
	if err != nil {
		return res, err
	}

	var sj records.StateJSON
	err = json.Unmarshal(b, &sj)
	if err != nil {
		return res, err
	}

	masters := []string{"144.76.157.37:5050"}
	res.rs = records.RecordGenerator{}
	res.rs.InsertState(sj, "mesos", "mesos-dns.mesos.", "127.0.0.1", masters)

	return res, nil
}

func fakeMsg(dom string, rrHeader uint16, proto string) (*dns.Msg, error) {
	qc := uint16(dns.ClassINET)

	c := new(dns.Client)
	c.Net = proto

	m := new(dns.Msg)
	m.Question = make([]dns.Question, 1)
	m.Question[0] = dns.Question{dns.Fqdn(dom), rrHeader, qc}

	in, _, err := c.Exchange(m, "127.0.0.1:8053")
	return in, err

}

func fakeQuery(dom string, rrHeader uint16, proto string) ([]dns.RR, error) {
	in, err := fakeMsg(dom, rrHeader, proto)
	if err != nil {
		return in.Answer, err
	}

	return in.Answer, nil
}

func identicalResults(msg_a []dns.RR, msg_b []dns.RR) bool {
	if len(msg_a) != len(msg_b) {
		return false
	}
	for i := range msg_a {
		if msg_a[i].String() != msg_b[i].String() {
			return false
		}
	}
	return true
}

func TestHandler(t *testing.T) {
	var msg []dns.RR

	res, err := fakeDNS(8053)
	if err != nil {
		t.Error(err)
	}

	dns.HandleFunc("mesos.", res.HandleMesos)
	go res.Serve("udp")
	go res.Serve("tcp")

	// wait for startup ? lame
	time.Sleep(10 * time.Millisecond)

	// test A records
	msg, err = fakeQuery("chronos.marathon-0.6.0.mesos.", dns.TypeA, "udp")
	if err != nil {
		t.Error(err)
	}

	if len(msg) != 1 {
		t.Error("not serving up A records")
	}

	// Test case sensitivity -- this test depends on one above
	msg_a := msg
	msg, err = fakeQuery("cHrOnOs.MARATHON-0.6.0.mesoS.", dns.TypeA, "udp")
	if err != nil {
		t.Error(err)
	}

	if !identicalResults(msg, msg_a) {
		t.Errorf("Case sensitivity failure:\n%s\n!=\n%s", msg, msg_a)
	}

	// test SRV record
	msg, err = fakeQuery("_liquor-store._udp.marathon-0.6.0.mesos.", dns.TypeSRV, "udp")
	if err != nil {
		t.Error(err)
	}

	if len(msg) != 3 {
		t.Error("not serving up SRV records")
	}

	// test SOA
	m, err2 := fakeMsg("non-existing.mesos.", dns.TypeSOA, "udp")
	if err2 != nil {
		t.Error(err2)
	}

	if m.Ns == nil {
		t.Error("not serving up SOA")
	}

	// test non-existing host
	m, err = fakeMsg("missing.mesos.", dns.TypeA, "udp")
	if err != nil {
		t.Error(err)
	}

	if m.Rcode != 3 {
		t.Error("not setting NXDOMAIN")
	}

	// test tcp
	msg, err = fakeQuery("chronos.marathon-0.6.0.mesos.", dns.TypeA, "tcp")
	if err != nil {
		t.Error(err)
	}

	if len(msg) != 1 {
		t.Error("not serving up A records")
	}

	// test AAAA --> NODATA
	m, err = fakeMsg("chronos.marathon-0.6.0.mesos.", dns.TypeAAAA, "udp")
	if err != nil {
		t.Error(err)
	}

	if m.Rcode != 0 || len(m.Answer) > 0 {
		t.Error("not setting NODATA for AAAA requests")
	}

	// test AAAA --> NXDOMAIN
	m, err = fakeMsg("missing.mesos.", dns.TypeAAAA, "udp")
	if err != nil {
		t.Error(err)
	}

	if m.Rcode != 3 {
		t.Error("not setting NXDOMAIN for AAAA requests")
	}

}

func TestNonMesosHandler(t *testing.T) {
	var msg []dns.RR

	res, err := fakeDNS(8054)
	if err != nil {
		t.Error(err)
	}

	dns.HandleFunc(".", res.HandleNonMesos)
	go res.Serve("udp")
	go res.Serve("tcp")

	// wait for startup ? lame
	time.Sleep(10 * time.Millisecond)

	// test A records
	msg, err = fakeQuery("google.com", dns.TypeA, "udp")
	if err != nil {
		t.Error(err)
	}

	if len(msg) < 2 {
		t.Error("not serving up A records")
	}

}
