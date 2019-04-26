package gyndns

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Config struct {
	HTTPAddress string
	HTTPPort    uint16

	DNSAddress string
	DNSPort    uint16
}

type Params struct {
	Config *Config
	Users  []User
}

type GynDNS struct {
	*Config

	users map[Username]User

	leases map[string]net.IP
	lMutex sync.RWMutex

	errChan chan error
}

type Username string

type User struct {
	Username Username
	Password string
	Names    []string
}

var defaultConfig = Config{
	HTTPAddress: "127.0.0.1",
	HTTPPort:    8000,
	DNSAddress:  "127.0.0.1",
	DNSPort:     5533,
}

func New(params *Params) *GynDNS {
	if params == nil {
		log.Fatal("Nil parametes supplied")
	}

	if params.Config == nil {
		params.Config = &defaultConfig
	}

	g := &GynDNS{
		Config:  params.Config,
		errChan: make(chan error),
		users:   make(map[Username]User),
		leases:  make(map[string]net.IP),
	}

	if len(params.Users) == 0 {
		log.Fatal("No users found in parameters file")
	}

	for _, u := range params.Users {
		g.users[u.Username] = u
	}

	return g
}

func (g *GynDNS) Run(ctxt context.Context) {
	ctxthttp, cancelhttp := context.WithCancel(ctxt)
	ctxtdns, canceldns := context.WithCancel(ctxt)

	cancel := func() {
		cancelhttp()
		canceldns()
	}

	go g.runHTTP(ctxthttp, g.errChan)
	go g.runDNS(ctxtdns, g.errChan)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-stop:
			cancel()
			break
		case err := <-g.errChan:
			panic(err)
		}
	}
}
