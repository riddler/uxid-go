package main

import (
	"fmt"
	"os"

	"github.com/riddler/uxid-go/v1"
	getopt "github.com/pborman/getopt/v2"
)

func main() {
	// getopt.HelpColumn = 50
	// getopt.DisplayWidth = 140

	fs := getopt.New()
	var (
		prefix = fs.StringLong("prefix", 'p', "", "prefix to use for generated UXID")
		size = fs.StringLong("size", 's', "10", "T-Shirt size (xs, s, m, l, xl) or (xsmall, small, medium, large, xlarge)")
		help = fs.BoolLong("help", 'h', "print this help text")
	)
	if err := fs.Getopt(os.Args, nil); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if *help {
		fs.PrintUsage(os.Stderr)
		os.Exit(0)
	}

	generate(*prefix, *size)
}

func generate(prefix string, size string) {
	id, err := uxid.Generate(prefix, size)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "%s\n", id)
}
