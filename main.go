package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gookit/slog"
)

func main() {
	var cfgFile, protocol string

	slog.Configure(func(logger *slog.SugaredLogger) {
		f := logger.Formatter.(*slog.TextFormatter)
		f.EnableColor = true
		f.SetTemplate("[{{datetime}}] [{{level}}] {{message}}\n")
	})

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: moko -protocol <%v> -cfg <cfg yaml file>\n", strings.Join(ServerMap.List(), ", "))
		flag.PrintDefaults()
	}
	flag.StringVar(&protocol, "protocol", "http", "mock server protocol")
	flag.StringVar(&cfgFile, "cfg", "", "mock configuration yaml")
	flag.Parse()

	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		slog.Fatalf("cfg file %v does not exist", cfgFile)
	}

	server, err := ServerMap.Get(protocol)
	if err != nil {
		slog.Fatal(err.Error())
	}

	if err = server.Init(cfgFile); err != nil {
		slog.Fatal(err)
	}
	slog.Fatal(server.Serve())

	// TODO: watch cfg file and reload server
}
