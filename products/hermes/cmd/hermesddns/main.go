package main

import (
	"log"

	"github.com/mca-rolando/HermesDDNS/internal/api"
	"github.com/mca-rolando/HermesDDNS/internal/config"
	"github.com/mca-rolando/HermesDDNS/internal/database"
	"github.com/mca-rolando/HermesDDNS/internal/ddns"
	"github.com/mca-rolando/HermesDDNS/internal/dns"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	if err := database.EnsureDomains(db, cfg.AllowedDomains, cfg.DefaultTTL, cfg.AllowWildcard); err != nil {
		log.Fatal(err)
	}

	updater := dns.NSUpdate{Binary: cfg.NSUpdateBinary, Server: cfg.DNSServer, TSIGKeyFile: cfg.TSIGKeyFile}
	service := &ddns.Service{DB: db, DNS: updater, AllowedDomains: cfg.AllowedDomains, DefaultTTL: cfg.DefaultTTL, AllowWildcard: cfg.AllowWildcard, AutocreatePolicy: cfg.AutocreatePolicy}
	server := api.New(db, cfg, service)
	log.Printf("HermesDDNS listening on %s", cfg.ListenAddress)
	server.Echo.Logger.Fatal(server.Echo.Start(cfg.ListenAddress))
}
