package main

import (
	"fmt"
	"os"
	"os/exec"
)

func cmdSetup() {
	tools := []struct {
		name string
		pkg  string
	}{
		{"govulncheck", "golang.org/x/vuln/cmd/govulncheck@latest"},
	}

	var failed int
	for _, t := range tools {
		fmt.Printf("installing %s ...\n", t.name)
		cmd := exec.Command("go", "install", t.pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "  failed: %v\n", err)
			failed++
			continue
		}
		fmt.Printf("  ok\n")
	}

	if failed > 0 {
		fmt.Fprintf(os.Stderr, "\n%d tool(s) failed to install\n", failed)
		os.Exit(1)
	}
	fmt.Println("\nall tools installed")
}
