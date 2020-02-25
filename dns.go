package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/gomodule/redigo/redis"
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
		switch q.Qtype {
		case dns.TypeMX:
			log.Printf("Searching for hostname '%s'", q.Name)
			response := &dns.Msg{
				MsgHdr: dns.MsgHdr{
					Id:            r.Id,
					Response:      true,
					Authoritative: true,
				},
			}

			response.Question = append(response.Question, q)

			r := g.pool.Get()
			defer r.Close()
			ip, err := redis.String(r.Do("GET",
				fmt.Sprintf("hostname/%s", strings.ToLower(q.Name))))

			if err != nil {
				response.Rcode = dns.RcodeNameError
				log.Printf("ERROR: %s\n", err)
				log.Println("Hostname " + q.Name + " not found in map")
			} else {
				response.Answer = append(response.Answer, &dns.MX{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeMX,
						Class:  dns.ClassINET,
						Ttl:    TTL,
					},
					Mx: strings.ToLower(q.Name),
				})
				log.Println(q.Name + " A " + ip)
			}

			rw.WriteMsg(response)
		/*
		case dns.TypeCAA:
			response := &dns.Msg{
				MsgHdr: dns.MsgHdr{
					Id:            r.Id,
					Response:      true,
					Authoritative: true,
				},
			}

			response.Question = append(response.Question, q)

			record := new(dns.CAA)
			record.Hdr = dns.RR_Header{
				Name: q.Name,
				Rrtype: dns.TypeCAA,
				Class: dns.ClassINET,
				Ttl: 0,
			}
			record.Tag = "issue"
			record.Value = "letsencrypt.org"
			//record.Flag = 1
			response.Answer = append(response.Answer, record)
			
			rw.WriteMsg(response)
		*/
		case dns.TypeA:
			log.Printf("Searching for hostname '%s'", q.Name)
			response := &dns.Msg{
				MsgHdr: dns.MsgHdr{
					Id:            r.Id,
					Response:      true,
					Authoritative: true,
				},
			}

			response.Question = append(response.Question, q)

			r := g.pool.Get()
			defer r.Close()
			ip, err := redis.String(r.Do("GET",
				fmt.Sprintf("hostname/%s", strings.ToLower(q.Name))))

			if err != nil {
				response.Rcode = dns.RcodeNameError
				log.Printf("ERROR: %s\n", err)
				log.Println("Hostname " + q.Name + " not found in map")
			} else {
				ipaddr := net.ParseIP(ip)
				response.Answer = append(response.Answer, &dns.A{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    TTL,
					},
					A: ipaddr,
				})
				log.Println(q.Name + " A " + ip)
			}

			rw.WriteMsg(response)
		default:
			log.Printf("Unsupported question type %d", q.Qtype)
			log.Printf("%+v\n", q)
			response := &dns.Msg{
				MsgHdr: dns.MsgHdr{
					Id:            r.Id,
					Response:      false,
					Authoritative: false,
				},
			}
			response.Rcode = dns.RcodeSuccess
			rw.WriteMsg(response)
			//response.Rcode = dns.RcodeNameError
		}
	}
}
