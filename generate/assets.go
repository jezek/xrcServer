package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	flag.Parse()
	if len(flag.Args()) != 2 {
		log.Fatalf("want exactly two arguments, got %d: %v", len(flag.Args()), flag.Args())
		os.Exit(1)
	}

	//dir := flag.Arg(0)
}
