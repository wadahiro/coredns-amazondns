package amazondns

import (
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

// AmazonDNS represents a plugin instance that can proxy requests to AmazonDNS.
type AmazonDNS struct {
	Next plugin.Handler

	client  *dns.Client
	zones   []string
	zoneMap map[string]*Zone
}

type Zone struct {
	zone string
	dns  string
	soa  dns.RR
	ns   []dns.RR
	nsa  []dns.RR
}

// ServeDNS implements plugin.Handler.
func (ad AmazonDNS) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	name := state.Name()

	key := plugin.Zones(ad.zones).Matches(name)
	if key == "" {
		return plugin.NextOrFailure(ad.Name(), ad.Next, ctx, w, r)
	}
	zone := ad.zoneMap[key]

	qtype := state.QType()

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative, m.RecursionAvailable, m.Compress = true, true, true

L:
	switch qtype {
	case dns.TypeA:
		for _, nsa := range zone.nsa {
			if name == nsa.Header().Name {
				m.Answer = []dns.RR{nsa}
				m.Ns = zone.ns
				m.Rcode = dns.RcodeSuccess
				break L
			}
		}
		fallthrough
	case dns.TypeAAAA:
		fallthrough
	case dns.TypeCNAME:
		// Need recursive mode for getting record from AmazonDNS
		r.MsgHdr.RecursionDesired = true
		resp, _, err := ad.client.Exchange(r, zone.dns)

		if err != nil {
			return dns.RcodeServerFailure, err
		}

		m.Answer = resp.Answer
		m.Rcode = resp.Rcode

		// Overwrite authority and additional section
		if len(m.Answer) > 0 {
			m.Ns = zone.ns
			m.Extra = zone.nsa
		} else {
			handleNotFound(zone, name, m)
		}
	case dns.TypeNS:
		if name == zone.soa.Header().Name {
			m.Answer = zone.ns
			m.Extra = zone.nsa
			m.Rcode = dns.RcodeSuccess
		} else {
			handleNotFound(zone, name, m)
		}
	case dns.TypeSOA:
		if name == zone.soa.Header().Name {
			m.Answer = []dns.RR{zone.soa}
			m.Ns = zone.ns
		} else {
			handleNotFound(zone, name, m)
		}
	default:
		handleNotFound(zone, name, m)
	}

	state.SizeAndDo(m)
	m, _ = state.Scrub(m)
	w.WriteMsg(m)
	return dns.RcodeSuccess, nil
}

func handleNotFound(zone *Zone, name string, m *dns.Msg) {
	m.Ns = []dns.RR{zone.soa}

	if name == zone.soa.Header().Name {
		m.Rcode = dns.RcodeSuccess
		return
	}
	for _, ns := range zone.ns {
		if name == ns.Header().Name {
			m.Rcode = dns.RcodeSuccess
			return
		}
	}
	for _, nsa := range zone.nsa {
		if name == nsa.Header().Name {
			m.Rcode = dns.RcodeSuccess
			return
		}
	}

	// Error
	m.Rcode = dns.RcodeNameError
}

func (ad *AmazonDNS) fillResponse(state request.Request, m *dns.Msg) {
}

// Name implements the Handler interface.
func (ad AmazonDNS) Name() string { return "amazondns" }
