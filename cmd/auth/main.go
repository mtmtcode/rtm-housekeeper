package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/mtmtcode/rtm-housekeeper/rtm"
)

func main() {
	apiKey := os.Getenv("RTM_API_KEY")
	sharedSecret := os.Getenv("RTM_SHARED_SECRET")

	if apiKey == "" || sharedSecret == "" {
		fmt.Fprintln(os.Stderr, "RTM_API_KEY and RTM_SHARED_SECRET must be set.")
		os.Exit(1)
	}

	client := rtm.NewClient(apiKey, sharedSecret, "")

	frob, err := client.GetFrob()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get frob: %v\n", err)
		os.Exit(1)
	}

	authURL := client.GetAuthURL(frob)
	fmt.Println("Open the following URL in your browser to authorize this app:")
	fmt.Println()
	fmt.Println("  ", authURL)
	fmt.Println()
	fmt.Print("After authorizing, press Enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	token, err := client.GetToken(frob)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Authentication successful!")
	fmt.Println("Set the following in mise.toml:")
	fmt.Printf("RTM_AUTH_TOKEN = \"%s\"\n", token)
}
