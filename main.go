package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/andriyg76/glog"

	"go-page-process/processor"
)

func main() {
	config := parseArgs()
	log.Info("Parsed arguments: %+v\n", config)
	proc := processor.New(config)
	proc.Process()
}

func parseArgs() processor.Config {
	config := processor.DefaultConfig()
	help := false

	flag.StringVar(&config.OutputPath, "output-path", config.OutputPath, "directory where pages will be written, by default [pages]")
	flag.StringVar(&config.OutputPath, "o", config.OutputPath, "short for --output-path")
	flag.StringVar(&config.DataPath, "data-path", config.DataPath, "directory where page data is stored, by default [data]")
	flag.StringVar(&config.DataPath, "d", config.DataPath, "short for --data-path")
	flag.StringVar(&config.SharedPath, "shared-path", config.SharedPath, "directory where shared data is stored, by default [shared]")
	flag.StringVar(&config.SharedPath, "s", config.SharedPath, "short for --shared-path")
	flag.StringVar(&config.TemplatesPath, "templates-path", config.TemplatesPath, "directory where templates are stored, by default [.processor/templates]")
	flag.StringVar(&config.TemplatesPath, "t", config.TemplatesPath, "short for --templates-path")
	flag.BoolVar(&help, "help", false, "Display help")
	flag.BoolVar(&help, "h", false, "Display help")

	flag.Usage = func() {
		output := flag.CommandLine.Output()
		fmt.Fprintf(output, "Usage: %s [options]\n\nOptions:\n", os.Args[0])
		fmt.Fprintf(output, "  -o, --output-path      Output directory (default: %s)\n", config.OutputPath)
		fmt.Fprintf(output, "  -d, --data-path        Data directory (default: %s)\n", config.DataPath)
		fmt.Fprintf(output, "  -s, --shared-path      Shared data directory (default: %s)\n", config.SharedPath)
		fmt.Fprintf(output, "  -t, --templates-path   Templates directory (default: %s)\n", config.TemplatesPath)
		fmt.Fprintln(output, "  -h, --help             Display help")
	}
	flag.Parse()

	if help {
		flag.Usage()
		os.Exit(0)
	}

	return config
}
