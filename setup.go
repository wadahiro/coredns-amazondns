package amazondns

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/dnsutil"

	"github.com/mholt/caddy"
	"github.com/miekg/dns"
)

const AMAZON_METADATA_URL = "http://169.254.169.254/latest/meta-data/"

func init() {
	caddy.RegisterPlugin("amazondns", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	ad := AmazonDNS{
		client:  dnsClient(),
		zones:   []string{},
		zoneMap: map[string]*Zone{},
	}

	for c.Next() {
		args := c.RemainingArgs()

		if len(args) != 1 && len(args) != 2 {
			return c.ArgErr()
		}

		key := plugin.Host(args[0]).Normalize()

		var dnsAddr string
		var soa dns.RR
		var ns []dns.RR
		var nsa []dns.RR

		if len(args) == 2 {
			dnsAddr = args[1]
		}

		for c.NextBlock() {
			switch c.Val() {
			case "soa":
				if !c.NextArg() {
					return c.ArgErr()
				}
				rr, err := dns.NewRR(c.Val())
				if err != nil {
					return err
				}
				soa = rr
			case "ns":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return c.ArgErr()
				}
				for _, arg := range args {
					rr, err := dns.NewRR(arg)
					if err != nil {
						return err
					}
					ns = append(ns, rr)
				}
			case "nsa":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return c.ArgErr()
				}
				for _, arg := range args {
					rr, err := dns.NewRR(arg)
					if err != nil {
						return err
					}
					nsa = append(nsa, rr)
				}
			default:
				return c.Errf("unknown property '%s'", c.Val())
			}
		}

		var err error
		if dnsAddr == "" {
			dnsAddr, err = resolveAmazonDNS()
			if err != nil {
				return err
			}
		}
		dnsAddr, err = dnsutil.ParseHostPort(dnsAddr, "53")
		if err != nil {
			return err
		}

		// Required check
		if soa == nil {
			return c.Errf("'soa' property requires")
		}
		if ns == nil {
			return c.Errf("'ns' property requires")
		}
		if nsa == nil {
			return c.Errf("'nsa' property requires")
		}

		ad.zones = append(ad.zones, key)
		ad.zoneMap[key] = &Zone{
			dns: dnsAddr,
			soa: soa,
			ns:  ns,
			nsa: nsa,
		}
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return ad
	})

	return nil
}

func dnsClient() *dns.Client {
	c := new(dns.Client)
	c.Net = "udp"
	c.ReadTimeout = 1 * time.Second
	c.WriteTimeout = 1 * time.Second
	return c
}

func resolveAmazonDNS() (string, error) {
	resp, err := http.Get(AMAZON_METADATA_URL + "network/interfaces/macs/")
	if err != nil {
		log.Printf("[ERROR] Cannot fetch /latest/meta-data/network/interfaces/macs/")
		return "", err
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	macs := strings.Split(string(b), "\n")

	resp, err = http.Get(AMAZON_METADATA_URL + "network/interfaces/macs/" + macs[0] + "vpc-ipv4-cidr-block/")
	if err != nil {
		log.Printf("[ERROR] Cannot fetch /latest/meta-data/network/interfaces/macs/%svpc-ipv4-cidr-block", macs[0])
		return "", err
	}

	defer resp.Body.Close()
	b, _ = ioutil.ReadAll(resp.Body)
	cidr := string(b)
	ip, _, err := net.ParseCIDR(cidr)

	if err != nil {
		log.Printf("[ERROR] Fetched invalid CIDR: %s", cidr)
		return "", err
	}
	ip = ip.To4()
	ip[3] += 2

	return ip.String(), nil
}
