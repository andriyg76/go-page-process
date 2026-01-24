package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/andriyg76/glog"

	"github.com/andriyg76/go-page-process/processor"
	"github.com/andriyg76/go-page-process/server"
)

func main() {
	config, serve := parseArgs()
	log.Info("Parsed arguments: %+v\n", config)
	if serve.Enabled {
		log.Info("Serve enabled (port=%d, listen=%q)\n", serve.Port, serve.Listen)
	}
	proc := processor.New(config)
	if err := proc.Process(); err != nil {
		log.Error("Processing failed: %v", err)
		os.Exit(1)
	}
	if serve.Enabled {
		srv, err := server.New(proc, server.Options{
			Port:   serve.Port,
			Listen: serve.Listen,
		})
		if err != nil {
			log.Error("Serve setup failed: %v", err)
			os.Exit(1)
		}
		if err := srv.Serve(); err != nil {
			log.Error("Serve failed: %v", err)
			os.Exit(1)
		}
	}
}

type serveArgs struct {
	Enabled bool
	Port    int
	Listen  string
}

func parseArgs() (processor.Config, serveArgs) {
	config := processor.DefaultConfig()
	serve := serveArgs{Port: 8080}
	help := false

	flag.StringVar(&config.OutputPath, "output-path", config.OutputPath, "directory where pages will be written, by default [pages]")
	flag.StringVar(&config.OutputPath, "o", config.OutputPath, "short for --output-path")
	flag.StringVar(&config.DataPath, "data-path", config.DataPath, "directory where page data is stored, by default [data]")
	flag.StringVar(&config.DataPath, "d", config.DataPath, "short for --data-path")
	flag.StringVar(&config.SharedPath, "shared-path", config.SharedPath, "directory where shared data is stored, by default [shared]")
	flag.StringVar(&config.SharedPath, "s", config.SharedPath, "short for --shared-path")
	flag.StringVar(&config.TemplatesPath, "templates-path", config.TemplatesPath, "directory where templates are stored, by default [.processor/templates]")
	flag.StringVar(&config.TemplatesPath, "t", config.TemplatesPath, "short for --templates-path")
	flag.StringVar(&config.RootPath, "root", config.RootPath, "base directory for relative paths, by default current working directory")
	flag.StringVar(&config.RootPath, "r", config.RootPath, "short for --root")
	flag.BoolVar(&serve.Enabled, "serve", serve.Enabled, "start HTTP server after rendering")
	flag.IntVar(&serve.Port, "port", serve.Port, "port for --serve (default: 8080)")
	flag.IntVar(&serve.Port, "p", serve.Port, "short for --port")
	flag.StringVar(&serve.Listen, "listen", serve.Listen, "listen address (host:port) overrides --port")
	flag.StringVar(&serve.Listen, "l", serve.Listen, "short for --listen")
	flag.BoolVar(&help, "help", false, "Display help")
	flag.BoolVar(&help, "h", false, "Display help")

	flag.Usage = func() {
		output := flag.CommandLine.Output()
		fmt.Fprintf(output, "Usage: %s [options]\n\nOptions:\n", os.Args[0])
		fmt.Fprintf(output, "  -o, --output-path      Output directory (default: %s)\n", config.OutputPath)
		fmt.Fprintf(output, "  -d, --data-path        Data directory (default: %s)\n", config.DataPath)
		fmt.Fprintf(output, "  -s, --shared-path      Shared data directory (default: %s)\n", config.SharedPath)
		fmt.Fprintf(output, "  -t, --templates-path   Templates directory (default: %s)\n", config.TemplatesPath)
		fmt.Fprintln(output, "  -r, --root             Base directory for relative paths (default: current working directory)")
		fmt.Fprintln(output, "      --serve            Start HTTP server after rendering")
		fmt.Fprintln(output, "  -p, --port             Port for --serve (default: 8080)")
		fmt.Fprintln(output, "  -l, --listen           Listen address (host:port) overrides --port")
		fmt.Fprintln(output, "  -h, --help             Display help")
	}
	flag.Parse()

	if help {
		flag.Usage()
		os.Exit(0)
	}

	return config, serve
}
