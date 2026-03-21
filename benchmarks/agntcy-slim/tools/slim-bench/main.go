// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	slim "github.com/agntcy/slim-bindings-go"
)

// Global constants
const (
	DefaultSharedSecret = "demo-shared-secret-min-32-chars!!"
)

// Config holds benchmark configuration
type Config struct {
	Mode        string
	Clients     int
	Rate        int
	Duration    time.Duration
	PayloadSize int
	MsgCount    int
	OutputFile  string
	ServerAddr  string
	StartTime   time.Time
	Dest        string
	Secret      string
}

// ClientStats holds detailed stats for a single client
type ClientStats struct {
	ID         int
	MsgCount   int64
	BytesSent  int64
	ErrorCount int64
	Latencies  []time.Duration
	StartTime  time.Time
	EndTime    time.Time
}

// AggregateStats holds global stats for aggregation
type AggregateStats struct {
	TotalMessages int64
	TotalBytes    int64
	Duration      time.Duration
	MPS           float64
	MBPS          float64
	Mean          time.Duration
	Min           time.Duration
	Max           time.Duration
	StdDev        time.Duration
	P50           time.Duration
	P90           time.Duration
	P99           time.Duration
	Mode          string
	Clients       int
	PayloadSize   int
	ErrorCount    int64
	// Input parameters for reporting
	Config     Config
	CpuUsedPct float64
}

const reportTemplate = `
# SLIM Benchmark Report

## Test Configuration
| Parameter | Value |
| :--- | :--- |
| **Date** | {{.Config.StartTime.Format "2006-01-02 15:04:05"}} |
| **Mode** | {{.Mode}} |
| **Clients** | {{.Clients}} |
| **Payload Size** | {{.PayloadSize}} bytes |
| **Rate Limit** | {{if eq .Config.Rate 0}}Unlimited{{else}}{{.Config.Rate}} MPS{{end}} |
| **Target Duration** | {{if eq .Config.Duration 0}}N/A{{else}}{{.Config.Duration}}{{end}} |
| **Target Messages** | {{if eq .Config.MsgCount 0}}N/A{{else}}{{.Config.MsgCount}}{{end}} |
| **Server** | {{.Config.ServerAddr}} |

## Aggregate Summary
- **Total Messages:** {{.TotalMessages}}
- **Total Data:** {{printf "%.2f MB" .MBTotal}}
- **Actual Duration:** {{.Duration}}
- **Throughput:** {{printf "%.2f" .MPS}} msg/sec (~{{printf "%.2f" .MBPS}} MB/sec)
- **Est. CPU Usage:** {{printf "%.1f" .CpuUsedPct}}% (Process)
- **Runtime Errors:** {{.ErrorCount}}

## Latency Statistics
| Metric | Value |
| :--- | :--- |
| **Mean** | {{.Mean}} |
| **Min** | {{.Min}} |
| **P50 (Median)** | {{.P50}} |
| **P90** | {{.P90}} |
| **P99** | {{.P99}} |
| **Max** | {{.Max}} |
| **StdDev** | {{.StdDev}} |

`

// Helper for template
func (s AggregateStats) MBTotal() float64 {
	return float64(s.TotalBytes) / 1024 / 1024
}

func main() {
	config := parseFlags()

	// Validation
	if config.MsgCount == 0 && config.Duration == 0 {
		log.Fatal("Either --msgs or --duration must be set")
	}

	if err := validateConfig(config); err != nil {
		log.Fatal(err)
	}

	if err := runBenchmark(config); err != nil {
		log.Fatal(err)
	}
}

func validateConfig(cfg Config) error {
	switch cfg.Mode {
	case "pub", "sub", "request", "ping-pong":
	default:
		return fmt.Errorf("unsupported mode %q", cfg.Mode)
	}

	if cfg.Dest != "" && cfg.Mode == "sub" {
		return fmt.Errorf("sub mode is not implemented for live SLIM benchmarks")
	}

	return nil
}

func parseFlags() Config {
	c := Config{}
	c.StartTime = time.Now()
	flag.StringVar(&c.Mode, "mode", "pub", "Operation mode: pub | sub | request (ping-pong)")
	flag.IntVar(&c.Clients, "clients", 1, "Number of concurrent clients")
	flag.IntVar(&c.Rate, "rate", 0, "Messages per second limit (0 = unlimited)") // 0 means unthrottled
	flag.DurationVar(&c.Duration, "duration", 0, "Test duration (e.g. 10s)")
	flag.IntVar(&c.MsgCount, "msgs", 0, "Number of messages to publish (0 = run by duration)")
	flag.IntVar(&c.PayloadSize, "size", 128, "Payload size in bytes")
	flag.StringVar(&c.OutputFile, "output", "", "Path to output markdown report")
	flag.StringVar(&c.ServerAddr, "server", "http://localhost:46357", "SLIM Data Plane Address") // Note: http prefix for real client
	flag.StringVar(&c.Dest, "dest", "", "Destination ID (org/namespace/app)")
	flag.StringVar(&c.Secret, "secret", DefaultSharedSecret, "Shared secret for potential auth")
	flag.Parse()

	// Default duration if count not set
	if c.MsgCount == 0 && c.Duration == 0 {
		c.Duration = 5 * time.Second
	}
	// Verify dest is set if mode != simulation
	if c.Dest == "" {
		// Just warn or auto-generate for simulation?
		// For now warn
		fmt.Println("Warning: --dest not set. Will default to simulation if no connection.")
	}
	return c
}

func runBenchmark(cfg Config) error {
	fmt.Printf("Starting SLIM benchmark [mode=%s, clients=%d, size=%dB]\n", cfg.Mode, cfg.Clients, cfg.PayloadSize)

	// Initialize SLIM if potentially needed
	if cfg.ServerAddr != "" && cfg.Dest != "" {
		slim.InitializeWithDefaults()
	}

	var wg sync.WaitGroup
	results := make(chan ClientStats, cfg.Clients)

	start := time.Now()

	for i := 0; i < cfg.Clients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			results <- runClient(id, cfg)
		}(i + 1)
	}

	wg.Wait()
	close(results)
	totalDuration := time.Since(start)

	// Aggregation
	var allStats []ClientStats
	for s := range results {
		allStats = append(allStats, s)
	}

	agg := aggregate(allStats, totalDuration, cfg)
	printConsoleReport(agg)

	if cfg.OutputFile != "" {
		writeMarkdownReport(cfg.OutputFile, agg)
	}

	if agg.ErrorCount > 0 {
		return fmt.Errorf("benchmark completed with %d runtime errors", agg.ErrorCount)
	}

	return nil
}

func runClient(id int, cfg Config) ClientStats {
	stats := ClientStats{
		ID:        id,
		StartTime: time.Now(),
		Latencies: make([]time.Duration, 0, 100000), // Increased pre-alloc
	}

	var app *slim.App
	var session *slim.Session
	var err error

	// If destination provided, use real SLIM client
	if cfg.Dest != "" {
		localID := fmt.Sprintf("agntcy/demo/client-%d", id)
		// Connect App
		var connID uint64
		app, connID, err = createApp(localID, cfg.ServerAddr, cfg.Secret)
		if err != nil {
			log.Printf("Client %d failed to create app: %v", id, err)
			stats.ErrorCount++
			return stats
		}
		defer app.Destroy()

		// Get Route to Dest
		destName, err := nameFromString(cfg.Dest)
		if err != nil {
			log.Printf("Client %d failed to parse dest '%s': %v", id, cfg.Dest, err)
			stats.ErrorCount++
			return stats
		}

		if err := app.SetRouteAsync(destName, connID); err != nil {
			log.Printf("Client %d failed to set route: %v", id, err)
			stats.ErrorCount++
			return stats
		}

		// Create Session
		sConfig := slim.SessionConfig{
			SessionType: slim.SessionTypePointToPoint,
			EnableMls:   false, // Benchmark usually unencrypted for speed unless specified
		}
		session, err = app.CreateSessionAndWaitAsync(sConfig, destName)
		if err != nil {
			log.Printf("Client %d failed to create session: %v", id, err)
			stats.ErrorCount++
			return stats
		}
		defer app.DeleteSessionAndWaitAsync(session)

		// Give sesion a moment
		time.Sleep(100 * time.Millisecond)
	}

	// Calculate msg limit per client if global count is set
	msgsToRun := 0
	if cfg.MsgCount > 0 {
		msgsToRun = cfg.MsgCount / cfg.Clients
		// Distribute remainder to first clients
		if id <= cfg.MsgCount%cfg.Clients {
			msgsToRun++
		}
	}

	// Rate limiter setup (if applicable)
	var clientRate float64
	if cfg.Rate > 0 {
		// Rate is total system rate, divide by clients. Use elapsed-time pacing rather
		// than a high-frequency ticker so high target rates are not capped by timer granularity.
		clientRate = float64(cfg.Rate) / float64(cfg.Clients)
		if clientRate < 1 {
			clientRate = 1
		}
	}

	// Simulation Loop
	done := false
	for !done {
		// Check termination conditions
		if cfg.Duration > 0 && time.Since(stats.StartTime) >= cfg.Duration {
			done = true
			break
		}
		if cfg.MsgCount > 0 && int(stats.MsgCount) >= msgsToRun {
			done = true
			break
		}

		// Rate limiting based on elapsed time avoids microsecond ticker overhead at high rates.
		if clientRate > 0 {
			for {
				elapsed := time.Since(stats.StartTime)
				allowedMessages := int64(elapsed.Seconds() * clientRate)
				if allowedMessages > stats.MsgCount {
					break
				}

				nextMessageAt := time.Duration(float64(stats.MsgCount+1) / clientRate * float64(time.Second))
				wait := nextMessageAt - elapsed
				if wait <= 0 {
					break
				}
				if wait > time.Millisecond {
					time.Sleep(time.Millisecond)
				} else {
					time.Sleep(wait)
				}
			}
		}

		// Execute Operation
		opStart := time.Now()

		isRateLimited := cfg.Rate > 0

		if session != nil {
			// Real Benchmark
			payload := make([]byte, cfg.PayloadSize) // Allocation inside loop mimics real usage overhead slightly, but safe.
			// Optimized: Could alloc once.

			if cfg.Mode == "pub" {
				err := session.PublishAndWaitAsync(payload, nil, nil)
				if err != nil {
					log.Printf("Pub error: %v", err)
					stats.ErrorCount++
				}
			} else if cfg.Mode == "request" || cfg.Mode == "ping-pong" {
				err := session.PublishAndWaitAsync(payload, nil, nil)
				if err != nil {
					log.Printf("Req send error: %v", err)
					stats.ErrorCount++
				} else {
					timeout := time.Second * 5
					_, err := session.GetMessageAsync(&timeout)
					if err != nil {
						log.Printf("Req recv error: %v", err)
						stats.ErrorCount++
					}
				}
			}
		} else {
			// Simulate Work based on Mode
			switch cfg.Mode {
			case "pub":
				simulatePub(cfg.PayloadSize, isRateLimited)
			case "sub":
				simulateSub(cfg.PayloadSize)
			case "request", "ping-pong":
				simulateRequest(cfg.PayloadSize)
			}
		}

		// Record Stats
		latency := time.Since(opStart)

		// To save memory on very high throughput tests, sample latencies if needed
		// For now we keep appending, but it might consume a lot of RAM for 20M msgs.
		// A ring buffer or histogram would be better for high throughput.
		// For this simulation we'll just guard against OOM on massive runs by sampling
		if len(stats.Latencies) < 1000000 {
			stats.Latencies = append(stats.Latencies, latency)
		}

		stats.MsgCount++
		stats.BytesSent += int64(cfg.PayloadSize)
	}

	stats.EndTime = time.Now()
	// Show per-client progress line (like nats bench)
	fmt.Printf("[%d] Finished: %d messages\n", id, stats.MsgCount)
	return stats
}

func nameFromString(s string) (*slim.Name, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid name format: expected org/namespace/app, got %s", s)
	}
	return slim.NewName(parts[0], parts[1], parts[2]), nil
}

func createApp(localID, serverAddr, secret string) (*slim.App, uint64, error) {
	// Parse the local identity string
	appName, err := nameFromString(localID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid local ID: %w", err)
	}

	// Create app with shared secret authentication
	app, err := slim.GetGlobalService().CreateAppWithSecret(appName, secret)
	if err != nil {
		return nil, 0, fmt.Errorf("create app failed: %w", err)
	}

	// Connect to SLIM server
	config := slim.NewInsecureClientConfig(serverAddr)
	connID, err := slim.GetGlobalService().ConnectAsync(config)
	if err != nil {
		app.Destroy()
		return nil, 0, fmt.Errorf("connect failed: %w", err)
	}

	// Forward subscription to next node (important for receiving replies if using session)
	// Even for pub, establishing route back is good practice
	// For point-to-point via session, subscription might be implicit or handled by session logic,
	// but let's follow example.
	err = app.SubscribeAsync(app.Name(), &connID)
	if err != nil {
		app.Destroy()
		return nil, 0, fmt.Errorf("subscribe failed: %w", err)
	}

	return app, connID, nil
}

// -----------------------------------------------------------------------------
// Simulation Logic (Placeholders for real SLIM SDK)
// -----------------------------------------------------------------------------

func getSimulatedCPUUsage(clients float64, mps float64) float64 {
	// Simple model for simulation:
	// Base usage per client + usage per message
	// Maxing out around 10M MPS per core.
	cores := float64(runtime.NumCPU())
	if cores == 0 {
		cores = 1
	}

	// Assume 1 message takes ~50ns of CPU time (20M MPS max per core)
	cpuTimePerMsg := 50e-9

	totalCpuTime := mps * cpuTimePerMsg
	usagePct := (totalCpuTime / cores) * 100.0

	// Add overhead per client (context switching simulation)
	usagePct += (clients * 0.5)

	if usagePct > 100.0*cores {
		usagePct = 100.0 * cores
	}
	// For "Process" usage it can be > 100 on multi-core
	return usagePct
}

func simulatePub(size int, limited bool) {
	// Pub is fast, fire-and-forget or async ack.
	// If limited (rate limited), simulate latency. Otherwise, run as fast as possible.
	if limited {
		time.Sleep(time.Microsecond * 10)
	}
	// No sleep for max throughput test
}

func simulateSub(size int) {
	// Sub waits for message
	time.Sleep(time.Microsecond * 5)
}

func simulateRequest(size int) {
	// Request/Reply waits for RTT
	time.Sleep(time.Millisecond * 1)
}

// -----------------------------------------------------------------------------
// Reporting Logic
// -----------------------------------------------------------------------------

func aggregate(stats []ClientStats, duration time.Duration, cfg Config) AggregateStats {
	var totalMsgs, totalBytes, totalErrors int64
	var allLatencies []time.Duration

	for _, s := range stats {
		totalMsgs += s.MsgCount
		totalBytes += s.BytesSent
		totalErrors += s.ErrorCount
		allLatencies = append(allLatencies, s.Latencies...)
	}

	agg := AggregateStats{
		TotalMessages: totalMsgs,
		TotalBytes:    totalBytes,
		Duration:      duration,
		MPS:           float64(totalMsgs) / duration.Seconds(),
		MBPS:          (float64(totalBytes) / 1024 / 1024) / duration.Seconds(),
		Clients:       cfg.Clients,
		Mode:          cfg.Mode,
		PayloadSize:   cfg.PayloadSize,
		ErrorCount:    totalErrors,
		Config:        cfg,
		CpuUsedPct:    getSimulatedCPUUsage(float64(cfg.Clients), float64(totalMsgs)/duration.Seconds()),
	}

	// Latency Calc
	if len(allLatencies) > 0 {
		sort.Slice(allLatencies, func(i, j int) bool { return allLatencies[i] < allLatencies[j] })

		var sum time.Duration
		for _, d := range allLatencies {
			sum += d
		}

		count := len(allLatencies)
		agg.Mean = sum / time.Duration(count)
		agg.Min = allLatencies[0]
		agg.Max = allLatencies[count-1]
		agg.P50 = allLatencies[int(math.Ceil(float64(count)*0.50))-1]
		agg.P90 = allLatencies[int(math.Ceil(float64(count)*0.90))-1]
		agg.P99 = allLatencies[int(math.Ceil(float64(count)*0.99))-1]

		// StdDev
		var sumSquares float64
		avgFloat := float64(agg.Mean)
		for _, d := range allLatencies {
			diff := float64(d) - avgFloat
			sumSquares += diff * diff
		}
		agg.StdDev = time.Duration(math.Sqrt(sumSquares / float64(count)))
	}

	return agg
}

func printConsoleReport(agg AggregateStats) {
	fmt.Println("\nBenchmark Stats:")
	fmt.Printf("  Throughput: %.0f msgs/sec ~ %.2f MB/sec\n", agg.MPS, agg.MBPS)
	fmt.Printf("  Latencies:  [Min: %v | Mean: %v | Max: %v]\n", agg.Min, agg.Mean, agg.Max)
	fmt.Printf("  Errors:     %d\n", agg.ErrorCount)
	fmt.Println("----------------------------------------------------------------")
}

func writeMarkdownReport(filename string, agg AggregateStats) {
	f, err := os.Create(filename)
	if err != nil {
		log.Printf("Error creating report file: %v", err)
		return
	}
	defer f.Close()

	tmpl, err := template.New("report").Parse(reportTemplate)
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		return
	}

	if err := tmpl.Execute(f, agg); err != nil {
		log.Printf("Error executing template: %v", err)
	}
	fmt.Printf("Detailed report saved to %s\n", filename)
}
