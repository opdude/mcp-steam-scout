// setup-psn validates a PSN NPSSO token and prints the config line
// to add it to the MCP server environment.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/opdude/mcp-steam-scout/internal/adapter"
)

func main() {
	npsso := flag.String("npsso", "", "Your PSN NPSSO token (from the npsso cookie on playstation.com)")
	flag.Parse()

	if *npsso == "" {
		fmt.Fprintln(os.Stderr, "Error: --npsso is required")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "To get your NPSSO token:")
		fmt.Fprintln(os.Stderr, "  1. Open https://ca.account.sony.com/api/v1/ssocookie in your browser")
		fmt.Fprintln(os.Stderr, "     while logged into your PlayStation account.")
		fmt.Fprintln(os.Stderr, "  2. Copy the npsso value from the JSON response.")
		fmt.Fprintln(os.Stderr, "  3. Run this tool with: setup-psn --npsso <token>")
		os.Exit(1)
	}

	fmt.Println("=== PSN NPSSO Validation ===")
	fmt.Println()
	fmt.Print("Validating NPSSO token... ")

	psn, err := adapter.NewPSNAdapter(*npsso)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAILED")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "The NPSSO token may be expired or invalid.")
		fmt.Fprintln(os.Stderr, "Get a fresh one from the npsso cookie on playstation.com.")
		os.Exit(1)
	}

	games, err := psn.GetLibrary()
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAILED")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Error fetching library: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("OK")
	fmt.Println()
	fmt.Printf("Successfully authenticated! Found %d games in your PSN library.\n", len(games))
	fmt.Println()
	fmt.Println("Add this to your opencode.json environment section:")
	fmt.Printf("  \"PSN_NPSSO\": \"%s\"\n", *npsso)
}
