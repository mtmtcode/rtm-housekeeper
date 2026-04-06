package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mtmtcode/rtm-housekeeper/housekeeper"
	"github.com/mtmtcode/rtm-housekeeper/rtm"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "show tasks that would be affected without making changes")
	flag.Parse()

	apiKey := os.Getenv("RTM_API_KEY")
	sharedSecret := os.Getenv("RTM_SHARED_SECRET")
	authToken := os.Getenv("RTM_AUTH_TOKEN")

	if apiKey == "" || sharedSecret == "" || authToken == "" {
		fmt.Fprintln(os.Stderr, "RTM_API_KEY, RTM_SHARED_SECRET, and RTM_AUTH_TOKEN must be set.")
		fmt.Fprintln(os.Stderr, "Set them in mise.toml, then run `go run ./cmd/auth` to get an auth token.")
		os.Exit(1)
	}

	if *dryRun {
		log.Println("[dry-run] mode enabled — no changes will be made")
	}

	client := rtm.NewClient(apiKey, sharedSecret, authToken)
	h := housekeeper.New(client, *dryRun)

	if err := h.Run(); err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Println("done")
}
