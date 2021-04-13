package main

import (
	"flag"
	"fmt"
	"os"
	"log"
	"strings"
)

func main() {
	var cfg, protocol string

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: moko -protocol <%v> -cfg <cfg yaml file>\n", strings.Join(ServerMap.List(), ", "))
		flag.PrintDefaults()
	}
	flag.StringVar(&protocol, "protocol", "http", "mock server protocol")
	flag.StringVar(&cfg, "cfg", "", "mock configuration yaml")
	flag.Parse()

	if _, err := os.Stat(cfg); os.IsNotExist(err) {
		log.Fatalf("cfg file %v does not exist", cfg)
	}

	server, err := ServerMap.Get(protocol)
	if err != nil {
		log.Fatal(err.Error())
	}

	if err = server.Init(cfg); err != nil {
		log.Fatal(err)
	}
	log.Fatal(server.Serve())
}
