package main

import (
	"flag"
	"os"

	log "github.com/andriyg76/glog"
)

type Params struct {
	OutputPath string
}

func main() {
	params := parseArgs()
	log.Info("Parsed arguments: %+v\n", params)
	processor := NewProcessor(params.OutputPath)
	processor.Process()
}

func parseArgs() Params {
	outputPath := flag.String("output-path", "pages", "directory where pages will be written, by default [pages]")
	help := flag.Bool("help", false, "Display help")
	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	return Params{
		OutputPath: *outputPath,
	}
}
