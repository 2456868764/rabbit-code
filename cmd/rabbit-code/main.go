package main

import (
	"fmt"
	"os"

	"github.com/2456868764/rabbit-code/internal/version"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("rabbit-code %s (%s)\n", version.Version, version.Commit)
		return
	}
	fmt.Fprintf(os.Stderr, "rabbit-code — Phase 0 scaffold. Use: rabbit-code version\n")
	os.Exit(1)
}
