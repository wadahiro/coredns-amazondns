# coredns-amazondns
The *amazondns* plugin behaves **Authoritative name server** using [Amazon DNS Server](https://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/VPC_DHCP_Options.html#AmazonDNS) as the backend.

The Amazon DNS server is used to resolve the DNS domain names that you specify in a private hosted zone in Route 53. However, the server acts as **Caching name server**. Although CoreDNS has [proxy plugin](https://github.com/coredns/coredns/tree/master/plugin/proxy) and we can configure Amazon DNS server as the backend, it can't be Authoritative name server. In my case, Authoritative name server is required to handle delegated responsibility for the subdomain. That's why I created this plugin. 

## Name

*amazondns* - enables serving Authoritative name server using Amazon DNS Server as the backend.

## Syntax

```txt
amazondns ZONE [ADDRESS] {
    soa RR
    ns RR
    nsa RR
}
```

* **ZONE** the zone scope for this plugin.
* **ADDRESS** defines the Amazon DNS server address specifically.
  If no **ADDRESS** entry, this plugin resolves it automatically using [Instance Metadata](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html).
* **soa** **RR** SOA record with [RFC 1035](https://tools.ietf.org/html/rfc1035#section-5) style.
* **ns** **RR** NS record(s) with [RFC 1035](https://tools.ietf.org/html/rfc1035#section-5) style.
* **nsa** **RR** A record(s) for the NS(s) with [RFC 1035](https://tools.ietf.org/html/rfc1035#section-5) style.
  The IP address will be that of the EC2 instance on which CoreDNS is running with this plugin.
  Note: You need to boot CoreDNS on an EC2 instance in the VPC because we can't access to Amazon DNS server from outside the VPC.

## Examples

Setup Route 53 as below.

* Create your Route 53 private hostead zone with `sub.example.org` and attach your VPC.
* Add A record as `test.sub.example.org` into the zone.
* Add CNAME record as `lb.sub.example.org` for your ELB into the zone.

Next, boot two EC2 instances for the name servers and deploy CoreDNS binary, and configure CoreDNS config file as below.

```txt
. {
    amazondns sub.example.org {
        soa "sub.example.org 3600 IN SOA ns1.sub.example.org hostmaster.sub.example.org (2018030619 3600 900 1209600 900)"
        ns "sub.example.org 3600 IN NS ns1.sub.example.org"
        ns "sub.example.org 3600 IN NS ns2.sub.example.org"
        nsa "ns1.sub.example.org 3600 IN A 192.168.0.10"
        nsa "ns2.sub.example.org 3600 IN A 192.168.0.130"
    }
}
```

Start CoreDNS and check it how it works.

The `test.sub.example.org` is resolved with *AUTHORITY SECTION* and *ADDITIONAL SECTION* as below.

```bash
> dig @localhost test.sub.example.org +norecurse

; <<>> DiG 9.11.1 <<>> @localhost test.sub.example.org +norecurse
; (1 server found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 28681
;; flags: qr aa ra; QUERY: 1, ANSWER: 1, AUTHORITY: 2, ADDITIONAL: 3

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
; COOKIE: 23246de45b4a3601 (echoed)
;; QUESTION SECTION:
;test.sub.example.org.        IN  A

;; ANSWER SECTION:
test.sub.example.org.   3600  IN  A  10.0.0.10

;; AUTHORITY SECTION:
sub.example.org.        3600  IN  NS  ns1.sub.example.org.
sub.example.org.        3600  IN  NS  ns2.sub.example.org.

;; ADDITIONAL SECTION:
ns1.sub.example.org.    3600  IN  A   192.168.0.10
ns2.sub.example.org.    3600  IN  A   192.168.0.130

;; Query time: 12 msec
;; SERVER: 127.0.0.1#53(127.0.0.1)
;; WHEN: Tue Feb 20 15:11:55 JST 2018
;; MSG SIZE  rcvd: 146
```

Also it can return NS record(s) for subdomain as below.

```bash
> dig @localhost sub.example.org ns

; <<>> DiG 9.11.1 <<>> @localhost sub.example.org ns +norecurse
; (1 server found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 2719
;; flags: qr aa ra; QUERY: 1, ANSWER: 2, AUTHORITY: 0, ADDITIONAL: 3

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
; COOKIE: c1c3332966dba8fd (echoed)
;; QUESTION SECTION:
;sub.example.org.            IN  NS

;; ANSWER SECTION:
sub.example.org.       3600  IN  NS  ns1.sub.example.org.
sub.example.org.       3600  IN  NS  ns2.sub.example.org.

;; ADDITIONAL SECTION:
ns1.sub.example.org.   3600  IN  A   192.168.0.10
ns2.sub.example.org.   3600  IN  A   192.168.0.130

;; Query time: 1 msec
;; SERVER: 127.0.0.1#53(127.0.0.1)
;; WHEN: Tue Feb 20 15:08:27 JST 2018
;; MSG SIZE  rcvd: 125
```

And it works like Route 53 alias record when answering CNAME record.
Your CNAME record will be removed and A/AAAA records of the CNAME record will be replaces.

```bash
> dig @localhost lb.sub.example.org

; <<>> DiG 9.11.1 <<>> @localhost test.sub.example.org +norecurse
; (1 server found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 63630
;; flags: qr aa ra; QUERY: 1, ANSWER: 2, AUTHORITY: 2, ADDITIONAL: 3

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
; COOKIE: 89a840e16b4d3fc7 (echoed)
;; QUESTION SECTION:
;lb.sub.example.org.         IN  A

;; ANSWER SECTION:
lb.sub.example.org.    54    IN  A   10.0.0.16
lb.sub.example.org.    54    IN  A   10.0.0.132

;; AUTHORITY SECTION:
sub.example.org.       3600  IN  NS  ns1.sub.example.org.
sub.example.org.       3600  IN  NS  ns2.sub.example.org.

;; ADDITIONAL SECTION:
ns1.sub.example.org.   3600  IN  A   192.168.0.10
ns2.sub.example.org.   3600  IN  A   192.168.0.130

;; Query time: 11 msec
;; SERVER: 127.0.0.1#53(127.0.0.1)
;; WHEN: Tue Feb 27 11:20:08 JST 2018
;; MSG SIZE  rcvd: 174
```
