package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func cmdUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	system := fs.Bool("system", false, "also update the system-wide binary and restart the system service")
	fs.Parse(args)

	fmt.Println("updating modex ...")
	cmd := exec.Command("go", "install", "github.com/0xCarbon/modex/cmd/modex@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "go install failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("binary updated")

	if *system {
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			home, _ := os.UserHomeDir()
			gopath = filepath.Join(home, "go")
		}
		src := filepath.Join(gopath, "bin", "modex")
		dst := "/usr/local/bin/modex"

		// Copy the binary (requires appropriate permissions, typically sudo).
		data, err := os.ReadFile(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read %s: %v\n", src, err)
			os.Exit(1)
		}
		if err := os.WriteFile(dst, data, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write %s: %v\n  (try running with sudo)\n", dst, err)
			os.Exit(1)
		}
		fmt.Printf("copied %s → %s\n", src, dst)
	}

	// Restart any detected service.
	for _, svc := range detectServices() {
		fmt.Printf("restarting %s ...\n", svc.name)
		cmd := svc.restartCmd()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "  restart failed: %v\n", err)
		} else {
			fmt.Println("  ok")
		}
	}
}

type detectedService struct {
	name   string
	system bool
}

func (d detectedService) restartCmd() *exec.Cmd {
	if d.system {
		return exec.Command("systemctl", "restart", d.name)
	}
	return exec.Command("systemctl", "--user", "restart", d.name)
}

func detectServices() []detectedService {
	var svcs []detectedService

	// Check user service.
	if checkServiceExists("modex.service", false) {
		svcs = append(svcs, detectedService{name: "modex.service", system: false})
	}
	// Check system service.
	if checkServiceExists("modex.service", true) {
		svcs = append(svcs, detectedService{name: "modex.service", system: true})
	}
	return svcs
}

func checkServiceExists(name string, system bool) bool {
	var cmd *exec.Cmd
	if system {
		cmd = exec.Command("systemctl", "cat", name)
	} else {
		cmd = exec.Command("systemctl", "--user", "cat", name)
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}
