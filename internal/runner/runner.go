// Package runner provides the core automation logic for AutoAR.
// It orchestrates reconnaissance and attack surface discovery workflows.
package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/autoar/internal/config"
)

// Runner holds the configuration and state for an AutoAR scan session.
type Runner struct {
	Config  *config.Config
	Logger  *log.Logger
	Results []Result
	mu      sync.Mutex
}

// Result represents the output of a single tool execution.
type Result struct {
	Tool      string
	Target    string
	Output    string
	Error     error
	StartedAt time.Time
	Finished  time.Time
}

// New creates a new Runner instance with the provided configuration.
func New(cfg *config.Config) (*Runner, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must not be nil")
	}

	logger := log.New(os.Stdout, "[autoar] ", log.LstdFlags|log.Lmsgprefix)

	return &Runner{
		Config:  cfg,
		Logger:  logger,
		Results: make([]Result, 0),
	}, nil
}

// Run starts the full automation pipeline for the given target domain.
func (r *Runner) Run(ctx context.Context, target string) error {
	if target == "" {
		return fmt.Errorf("target domain must not be empty")
	}

	target = strings.TrimSpace(strings.ToLower(target))
	r.Logger.Printf("Starting AutoAR scan for target: %s", target)

	// Ensure output directory exists
	outDir := filepath.Join(r.Config.OutputDir, target)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outDir, err)
	}

	phases := []struct {
		name string
		fn   func(context.Context, string, string) error
	}{
		{"subdomain-enum", r.runSubdomainEnumeration},
		{"port-scan", r.runPortScan},
		{"web-probe", r.runWebProbe},
	}

	for _, phase := range phases {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		r.Logger.Printf("Running phase: %s", phase.name)
		if err := phase.fn(ctx, target, outDir); err != nil {
			r.Logger.Printf("Phase %s encountered error: %v", phase.name, err)
			// Continue to next phase unless context is cancelled
		}
	}

	r.Logger.Printf("Scan completed for target: %s", target)
	return nil
}

// runSubdomainEnumeration executes subdomain discovery tools against the target.
func (r *Runner) runSubdomainEnumeration(ctx context.Context, target, outDir string) error {
	outFile := filepath.Join(outDir, "subdomains.txt")
	return r.execTool(ctx, "subfinder", []string{"-d", target, "-o", outFile, "-silent"}, target)
}

// runPortScan performs port scanning on discovered hosts.
func (r *Runner) runPortScan(ctx context.Context, target, outDir string) error {
	inFile := filepath.Join(outDir, "subdomains.txt")
	if _, err := os.Stat(inFile); os.IsNotExist(err) {
		r.Logger.Printf("Skipping port scan: subdomains file not found at %s", inFile)
		return nil
	}
	outFile := filepath.Join(outDir, "ports.txt")
	return r.execTool(ctx, "naabu", []string{"-list", inFile, "-o", outFile, "-silent"}, target)
}

// runWebProbe probes discovered hosts for active HTTP/HTTPS services.
func (r *Runner) runWebProbe(ctx context.Context, target, outDir string) error {
	inFile := filepath.Join(outDir, "subdomains.txt")
	if _, err := os.Stat(inFile); os.IsNotExist(err) {
		r.Logger.Printf("Skipping web probe: subdomains file not found at %s", inFile)
		return nil
	}
	outFile := filepath.Join(outDir, "alive.txt")
	return r.execTool(ctx, "httpx", []string{"-list", inFile, "-o", outFile, "-silent"}, target)
}

// execTool runs an external command and records its result.
func (r *Runner) execTool(ctx context.Context, tool string, args []string, target string) error {
	start := time.Now()
	result := Result{
		Tool:      tool,
		Target:    target,
		StartedAt: start,
	}

	cmd := exec.CommandContext(ctx, tool, args...)
	out, err := cmd.CombinedOutput()
	result.Output = string(out)
	result.Error = err
	result.Finished = time.Now()

	r.mu.Lock()
	r.Results = append(r.Results, result)
	r.mu.Unlock()

	if err != nil {
		return fmt.Errorf("tool %s failed: %w", tool, err)
	}
	return nil
}
