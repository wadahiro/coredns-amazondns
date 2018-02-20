package amazondns

import (
	"testing"

	"github.com/mholt/caddy"
)

func TestSetupRoute53(t *testing.T) {

	c := caddy.NewTestController("dns", `amazondns`)
	if err := setup(c); err == nil {
		t.Fatalf("Expected errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `amazondns example.org`)
	if err := setup(c); err == nil {
		t.Fatalf("Expected errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `amazondns example.org 10.0.0.2`)
	if err := setup(c); err == nil {
		t.Fatalf("Expected errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `amazondns example.org 10.0.0.2 {
    soa "example.org 60 IN SOA ns1.example.org hostmaster.example.org (1 7200 900 1209600 86400)"
}`)
	if err := setup(c); err == nil {
		t.Fatalf("Expected errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `amazondns example.org 10.0.0.2 {
    soa "example.org 60 IN SOA ns1.example.org hostmaster.example.org (1 7200 900 1209600 86400)"
    ns "example.org 60 IN NS ns1.example.org"
}`)
	if err := setup(c); err == nil {
		t.Fatalf("Expected errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `amazondns example.org 10.0.0.2 {
    soa "example.org 60 IN SOA ns1.example.org hostmaster.example.org (1 7200 900 1209600 86400)"
    ns "example.org 60 IN NS ns1.example.org"
    nsa "ns1.example.org 60 IN A 192.168.0.1"
}`)
	if err := setup(c); err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `amazondns example.org 10.0.0.2 {
    soa "example.org 60 IN SOA ns1.example.org hostmaster.example.org (1 7200 900 1209600 86400)"
    ns "example.org 60 IN NS ns1.example.org"
    ns "example.org 60 IN NS ns2.example.org"
    nsa "ns1.example.org 60 IN A 192.168.0.1"
    nsa "ns2.example.org 60 IN A 192.168.0.2"
}`)
	if err := setup(c); err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `amazondns example.org invalid {
    soa "example.org 60 IN SOA ns1.example.org hostmaster.example.org (1 7200 900 1209600 86400)"
    ns "example.org 60 IN NS ns1.example.org"
    nsa "ns1.example.org 60 IN A 192.168.0.1"
}`)
	if err := setup(c); err == nil {
		t.Fatalf("Expected errors, but got: %v", err)
	}
}
