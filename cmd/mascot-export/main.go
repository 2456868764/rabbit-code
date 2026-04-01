// Mascot-export writes assets/rabbit-code-mascot.png (same raster as the CLI splash).
// Run from module root: go run ./cmd/mascot-export
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/2456868764/rabbit-code/internal/app"
)

func main() {
	b, err := app.MascotPNG()
	if err != nil {
		log.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	out := filepath.Join(wd, "assets", "rabbit-code-mascot.png")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s (%d bytes)", out, len(b))
}
