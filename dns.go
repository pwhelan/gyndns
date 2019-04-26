package gyndns

import (
	"context"
	"fmt"
	"log"

	"github.com/miekg/dns"
)

const TTL = 16

func (g *GynDNS) runDNS(ctxt context.Context, errChan chan error) {
	addr := fmt.Sprintf("%s:%d", g.DNSAddress, g.DNSPort)
	log.Printf("Starting DNS server at %s...", addr)
	srv := &dns.Server{Addr: addr, Net: "udp", Handler: g}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			errChan <- err
		}
	}()
	go func() {
		<-ctxt.Done()
		srv.Shutdown()
	}()
}

func (g *GynDNS) ServeDNS(rw dns.ResponseWriter, r *dns.Msg) {
	for _, q := range r.Question {
		if q.Qtype != dns.TypeA {
			log.Printf("Unsupported question type %d", q.Qtype)
		} else {
			log.Printf("Searching for hostname '%s'", q.Name)
			response := &dns.Msg{
				MsgHdr: dns.MsgHdr{
					Id:            r.Id,
					Response:      true,
					Authoritative: true,
				},
			}

			response.Question = append(response.Question, q)

			g.lMutex.RLock()
			ip, found := g.leases[q.Name]
			g.lMutex.RUnlock()

			if found {
				response.Answer = append(response.Answer, &dns.A{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    TTL,
					},
					A: ip,
				})
				log.Println(q.Name + " A " + ip.String())
			} else {
				response.Rcode = dns.RcodeNameError
				log.Println("Hostname " + q.Name + " not found in map")
			}

			rw.WriteMsg(response)

			break
		}
	}
}
