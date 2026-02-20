package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
)

const unitTemplate = `[Unit]
Description=modex — Go module expert MCP server
After=network.target

[Service]
Type=simple
ExecStart={{.ExecStart}}
Restart=on-failure
RestartSec=5
Environment=PATH={{.Path}}
{{- if .RunAs}}
User={{.RunAs}}
{{- end}}

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
PrivateDevices=true
ProtectControlGroups=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectClock=true
RestrictSUIDSGID=true
LockPersonality=true
RestrictRealtime=true
RestrictNamespaces=true
SystemCallArchitectures=native
ProtectSystem=full
UMask=0077
LimitNOFILE=16384
TasksMax=512
TimeoutStartSec=120
TimeoutStopSec=30

[Install]
WantedBy={{.WantedBy}}
`

type unitData struct {
	ExecStart string
	Path      string
	RunAs     string
	WantedBy  string
}

func cmdInstallService(args []string) {
	fs := flag.NewFlagSet("install-service", flag.ExitOnError)
	host := fs.String("host", "127.0.0.1", "bind host")
	port := fs.String("port", "3838", "bind port")
	system := fs.Bool("system", false, "install as system-wide service (requires root)")
	runAs := fs.String("run-as", "", "user to run as (system mode only)")
	allowRemote := fs.Bool("allow-remote", false, "allow binding to non-loopback addresses")
	linger := fs.Bool("linger", false, "enable loginctl linger for user services")
	transport := fs.String("transport", "http", "transport: http or stdio")
	dbPath := fs.String("db", "", "database path (default: auto)")
	rateLimit := fs.Int("rate-limit", DefaultRateLimitPerSecond, "requests per second")
	maxConcurrent := fs.Int("max-concurrent", DefaultMaxConcurrent, "max concurrent requests")
	fs.Parse(args)

	addr := net.JoinHostPort(*host, *port)

	// Validate loopback unless --allow-remote.
	if !*allowRemote && !isLoopback(*host) {
		fmt.Fprintf(os.Stderr, "error: %q is not a loopback address\n", *host)
		fmt.Fprintf(os.Stderr, "  use --allow-remote to bind to non-loopback addresses\n")
		fmt.Fprintf(os.Stderr, "  WARNING: modex has no authentication\n")
		os.Exit(1)
	}

	// Resolve binary path.
	binPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve binary path: %v\n", err)
		os.Exit(1)
	}
	binPath, _ = filepath.EvalSymlinks(binPath)

	// For system mode, copy binary to /usr/local/bin.
	if *system {
		dst := "/usr/local/bin/modex"
		data, err := os.ReadFile(binPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read %s: %v\n", binPath, err)
			os.Exit(1)
		}
		if err := os.WriteFile(dst, data, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write %s: %v\n  (try running with sudo)\n", dst, err)
			os.Exit(1)
		}
		fmt.Printf("copied binary to %s\n", dst)
		binPath = dst
	}

	// Build ExecStart line.
	execArgs := []string{binPath, "server", "-transport", *transport, "-addr", addr}
	if *rateLimit != DefaultRateLimitPerSecond {
		execArgs = append(execArgs, "-rate-limit", fmt.Sprintf("%d", *rateLimit))
	}
	if *maxConcurrent != DefaultMaxConcurrent {
		execArgs = append(execArgs, "-max-concurrent", fmt.Sprintf("%d", *maxConcurrent))
	}
	if *dbPath != "" {
		execArgs = append(execArgs, "-db", *dbPath)
	}
	execStart := strings.Join(execArgs, " ")

	// Resolve Go toolchain PATH.
	envPath := resolveGoPath()

	wantedBy := "default.target"
	if *system {
		wantedBy = "multi-user.target"
	}

	data := unitData{
		ExecStart: execStart,
		Path:      envPath,
		WantedBy:  wantedBy,
	}
	if *system && *runAs != "" {
		data.RunAs = *runAs
	}

	// Render unit file.
	tmpl := template.Must(template.New("unit").Parse(unitTemplate))
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		fmt.Fprintf(os.Stderr, "failed to render unit: %v\n", err)
		os.Exit(1)
	}
	unitContent := buf.String()

	// Determine unit path.
	unitPath := systemUnitPath(*system)

	// Ensure directory exists.
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create directory: %v\n", err)
		os.Exit(1)
	}

	// Write unit file.
	if err := os.WriteFile(unitPath, []byte(unitContent), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", unitPath, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", unitPath)

	// Enable linger if requested.
	if *linger && !*system {
		u, err := user.Current()
		if err == nil {
			cmd := exec.Command("loginctl", "enable-linger", u.Username)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "warning: enable-linger failed: %v\n", err)
			} else {
				fmt.Println("enabled linger")
			}
		}
	}

	// daemon-reload, enable, start.
	systemctlRun(*system, "daemon-reload")
	systemctlRun(*system, "enable", "modex.service")
	systemctlRun(*system, "start", "modex.service")

	fmt.Println("modex service installed and started")
}

func cmdRemoveService(args []string) {
	fs := flag.NewFlagSet("remove-service", flag.ExitOnError)
	system := fs.Bool("system", false, "remove system-wide service")
	fs.Parse(args)

	// Stop and disable.
	systemctlRun(*system, "stop", "modex.service")
	systemctlRun(*system, "disable", "modex.service")

	// Remove unit file.
	unitPath := systemUnitPath(*system)
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "failed to remove %s: %v\n", unitPath, err)
	} else {
		fmt.Printf("removed %s\n", unitPath)
	}

	systemctlRun(*system, "daemon-reload")
	fmt.Println("modex service removed")
}

func systemUnitPath(system bool) string {
	if system {
		return "/etc/systemd/system/modex.service"
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", "modex.service")
}

func systemctlRun(system bool, args ...string) {
	var cmd *exec.Cmd
	if system {
		cmd = exec.Command("systemctl", args...)
	} else {
		fullArgs := append([]string{"--user"}, args...)
		cmd = exec.Command("systemctl", fullArgs...)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "systemctl %s: %v\n", strings.Join(args, " "), err)
	}
}

func isLoopback(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1", "[::1]":
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func resolveGoPath() string {
	parts := []string{}

	// GOPATH/bin
	gopath, err := exec.Command("go", "env", "GOPATH").Output()
	if err == nil {
		p := strings.TrimSpace(string(gopath))
		if p != "" {
			parts = append(parts, filepath.Join(p, "bin"))
		}
	}

	// GOROOT/bin
	goroot, err := exec.Command("go", "env", "GOROOT").Output()
	if err == nil {
		p := strings.TrimSpace(string(goroot))
		if p != "" {
			parts = append(parts, filepath.Join(p, "bin"))
		}
	}

	// System PATH.
	if sysPath := os.Getenv("PATH"); sysPath != "" {
		parts = append(parts, sysPath)
	}

	return strings.Join(parts, ":")
}
