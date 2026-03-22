// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	slim "github.com/agntcy/slim-bindings-go"
)

const defaultSharedSecret = "demo-shared-secret-min-32-chars!!"

var (
	warmupPayload   = []byte("__slim-bench-warmup__")
	warmupAckPrefix = []byte("__slim-bench-warmup-ack__:")
	echoAckPrefix   = []byte("__slim-bench-echo-ack__:")
	drainPayload    = []byte("__slim-bench-drain__:")
	drainAckPrefix  = []byte("__slim-bench-drain-ack__:")
)

type config struct {
	LocalID     string
	DestID      string
	Server      string
	Secret      string
	Mode        string
	Clients     int
	ShardDest   bool
	MaxInFlight int
	PayloadSize int
	Rate        int
	MsgCount    int
	Duration    time.Duration
	OutputFile  string
	StartTime   time.Time
}

type runStats struct {
	ID        int
	Sent      int64
	Bytes     int64
	Errors    int64
	Started   time.Time
	Finished  time.Time
	Min       time.Duration
	Max       time.Duration
	Total     time.Duration
	Latencies []time.Duration
}

type aggregateStats struct {
	TotalMessages int64
	TotalBytes    int64
	Duration      time.Duration
	MPS           float64
	MBPS          float64
	Mean          time.Duration
	Min           time.Duration
	Max           time.Duration
	P50           time.Duration
	P90           time.Duration
	P99           time.Duration
	StdDev        time.Duration
	ErrorCount    int64
	Config        config
}

func canonicalClientMode(mode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "request-reply":
		return "request-reply", nil
	case "fire-and-forget":
		return "fire-and-forget", nil
	case "write":
		return "write", nil
	default:
		return "", fmt.Errorf("unsupported mode %q", mode)
	}
}

const reportTemplate = `
# SLIM Rate Client Report

## Test Configuration
| Parameter | Value |
| :--- | :--- |
| **Date** | {{.Config.StartTime.Format "2006-01-02 15:04:05"}} |
| **Mode** | {{.Config.Mode}} |
| **Clients** | {{.Config.Clients}} |
| **Payload Size** | {{.Config.PayloadSize}} bytes |
| **Rate Limit** | {{.Config.Rate}} MPS |
| **Target Duration** | {{if eq .Config.MsgCount 0}}{{.Config.Duration}}{{else}}N/A{{end}} |
| **Target Messages** | {{if eq .Config.MsgCount 0}}N/A{{else}}{{.Config.MsgCount}}{{end}} |
| **Server** | {{.Config.Server}} |
| **Destination** | {{.Config.DestID}} |

## Aggregate Summary
- **Total Messages:** {{.TotalMessages}}
- **Total Data:** {{printf "%.2f MB" .MBTotal}}
- **Actual Duration:** {{.Duration}}
- **Throughput:** {{printf "%.2f" .MPS}} msg/sec (~{{printf "%.2f" .MBPS}} MB/sec)
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

func (s aggregateStats) MBTotal() float64 {
	return float64(s.TotalBytes) / 1024 / 1024
}

func main() {
	cfg := parseFlags()
	if err := validateConfig(cfg); err != nil {
		log.Fatal(err)
	}

	slim.InitializeWithDefaults()
	agg, err := runBenchmark(cfg)
	if err != nil {
		log.Fatal(err)
	}

	printConsoleReport(agg)
	if cfg.OutputFile != "" {
		if err := writeMarkdownReport(cfg.OutputFile, agg); err != nil {
			log.Fatal(err)
		}
	}

	if agg.ErrorCount > 0 {
		log.Fatalf("run completed with %d errors", agg.ErrorCount)
	}
}

func runBenchmark(cfg config) (aggregateStats, error) {
	var wg sync.WaitGroup
	results := make(chan runStats, cfg.Clients)
	start := time.Now()
	connID, err := connectToServer(cfg.Server)
	if err != nil {
		return aggregateStats{}, err
	}

	for index := 1; index <= cfg.Clients; index++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			results <- runClient(id, cfg, connID)
		}(index)
	}

	wg.Wait()
	close(results)

	stats := make([]runStats, 0, cfg.Clients)
	for result := range results {
		stats = append(stats, result)
	}

	return aggregate(stats, time.Since(start), cfg), nil
}

func runClient(id int, cfg config, connID uint64) runStats {
	stats := runStats{
		ID:        id,
		Started:   time.Now(),
		Min:       time.Duration(1<<63 - 1),
		Latencies: make([]time.Duration, 0, 100000),
	}

	app, err := createSubscribedApp(localIDForClient(cfg.LocalID, id, cfg.Clients), cfg.Secret, connID)
	if err != nil {
		log.Printf("client %d failed to create app: %v", id, err)
		stats.Errors++
		stats.Finished = time.Now()
		stats.Min = 0
		return stats
	}
	defer app.Destroy()

	destName, err := nameFromString(destIDForClient(cfg.DestID, id, cfg.Clients, cfg.ShardDest))
	if err != nil {
		log.Printf("client %d failed to parse destination: %v", id, err)
		stats.Errors++
		stats.Finished = time.Now()
		stats.Min = 0
		return stats
	}

	if err := app.SetRoute(destName, connID); err != nil {
		log.Printf("client %d failed to set route: %v", id, err)
		stats.Errors++
		stats.Finished = time.Now()
		stats.Min = 0
		return stats
	}

	session, err := app.CreateSessionAndWait(slim.SessionConfig{
		SessionType: slim.SessionTypePointToPoint,
		EnableMls:   false,
	}, destName)
	if err != nil {
		log.Printf("client %d failed to create session: %v", id, err)
		stats.Errors++
		stats.Finished = time.Now()
		stats.Min = 0
		return stats
	}
	defer func() {
		if err := app.DeleteSessionAndWait(session); err != nil {
			log.Printf("session cleanup failed: %v", err)
		}
	}()

	payload := make([]byte, cfg.PayloadSize)
	fmt.Printf("client=%d ready conn_id=%d app_id=%d\n", id, connID, app.Id())
	if cfg.Mode == "request-reply" {
		if err := warmupSession(session, id); err != nil {
			log.Printf("client %d warmup failed: %v", id, err)
			stats.Errors++
			stats.Finished = time.Now()
			stats.Min = 0
			return stats
		}
	}
	return sendSeries(session, payload, cfg, stats)
}

func parseFlags() config {
	var cfg config
	cfg.StartTime = time.Now()
	flag.StringVar(&cfg.LocalID, "local", "agntcy/demo/client", "Local ID in org/namespace/app format")
	flag.StringVar(&cfg.DestID, "dest", "agntcy/demo/echo", "Destination ID in org/namespace/app format")
	flag.StringVar(&cfg.Server, "server", "http://127.0.0.1:46357", "SLIM server endpoint")
	flag.StringVar(&cfg.Secret, "shared-secret", defaultSharedSecret, "Shared secret")
	flag.StringVar(&cfg.Mode, "mode", "fire-and-forget", "Send mode: request-reply | fire-and-forget | write")
	flag.IntVar(&cfg.Clients, "clients", 1, "Number of concurrent long-lived clients")
	flag.BoolVar(&cfg.ShardDest, "dest-sharded", false, "Distribute client sessions across destination identities with numeric suffixes")
	flag.IntVar(&cfg.MaxInFlight, "max-inflight", 1024, "Maximum number of outstanding async publishes per client in pub mode")
	flag.IntVar(&cfg.PayloadSize, "size", 128, "Payload size in bytes")
	flag.IntVar(&cfg.Rate, "rate", 1000, "Target messages per second")
	flag.IntVar(&cfg.MsgCount, "msgs", 0, "Number of messages to send (0 = run by duration)")
	flag.DurationVar(&cfg.Duration, "duration", 5*time.Second, "Run duration when -msgs=0")
	flag.StringVar(&cfg.OutputFile, "output", "", "Path to an output markdown report")
	flag.Parse()
	canonicalMode, err := canonicalClientMode(cfg.Mode)
	if err != nil {
		log.Fatal(err)
	}
	cfg.Mode = canonicalMode
	return cfg
}

func validateConfig(cfg config) error {
	if cfg.LocalID == "" || cfg.DestID == "" {
		return fmt.Errorf("both -local and -dest must be set")
	}
	canonicalMode, err := canonicalClientMode(cfg.Mode)
	if err != nil {
		return err
	}
	cfg.Mode = canonicalMode
	if cfg.Clients < 1 {
		return fmt.Errorf("clients must be >= 1")
	}
	if cfg.PayloadSize < 0 {
		return fmt.Errorf("payload size must be >= 0")
	}
	if cfg.MaxInFlight < 1 {
		return fmt.Errorf("max-inflight must be >= 1")
	}
	if cfg.Rate < 0 {
		return fmt.Errorf("rate must be >= 0")
	}
	if cfg.Mode == "request-reply" && cfg.Rate == 0 {
		return fmt.Errorf("rate must be > 0 for %s mode", cfg.Mode)
	}
	if cfg.MsgCount == 0 && cfg.Duration <= 0 {
		return fmt.Errorf("duration must be > 0 when msgs is 0")
	}
	return nil
}

func sendSeries(session *slim.Session, payload []byte, cfg config, stats runStats) runStats {
	stats.Started = time.Now()
	clientRate := 0.0
	if cfg.Rate > 0 {
		clientRate = float64(cfg.Rate) / float64(cfg.Clients)
		if clientRate < 1 {
			clientRate = 1
		}
	}

	msgsToRun := 0
	if cfg.MsgCount > 0 {
		msgsToRun = cfg.MsgCount / cfg.Clients
		if stats.ID <= cfg.MsgCount%cfg.Clients {
			msgsToRun++
		}
	}

	for {
		if msgsToRun > 0 && int(stats.Sent) >= msgsToRun {
			break
		}
		if cfg.MsgCount == 0 && time.Since(stats.Started) >= cfg.Duration {
			break
		}

		if clientRate > 0 {
			waitForTurn(stats.Started, stats.Sent, clientRate)
		}

		opStart := time.Now()
		var err error
		err = session.PublishAndWait(payload, nil, nil)
		if err == nil && cfg.Mode == "request-reply" {
			err = awaitEchoReply(session, payload, 5*time.Second)
		}

		latency := time.Since(opStart)
		if latency < stats.Min {
			stats.Min = latency
		}
		if latency > stats.Max {
			stats.Max = latency
		}
		stats.Total += latency

		if err != nil {
			stats.Errors++
			log.Printf("client %d send failed after %d messages: %v", stats.ID, stats.Sent, err)
			break
		}

		stats.Sent++
		stats.Bytes += int64(len(payload))
		if len(stats.Latencies) < 1000000 {
			stats.Latencies = append(stats.Latencies, latency)
		}
	}

	stats.Finished = time.Now()
	if stats.Sent == 0 {
		stats.Min = 0
	}
	return stats
}

func warmupSession(session *slim.Session, clientID int) error {
	if err := session.PublishAndWait(warmupPayload, nil, nil); err != nil {
		return fmt.Errorf("publish warmup: %w", err)
	}

	payload, err := awaitMatchingMessage(session, 5*time.Second, func(payload []byte) bool {
		return bytes.HasPrefix(payload, warmupAckPrefix)
	})
	if err != nil {
		return fmt.Errorf("await warmup reply: %w", err)
	}
	if !bytes.HasPrefix(payload, warmupAckPrefix) {
		return fmt.Errorf("unexpected warmup reply payload for client %d", clientID)
	}

	return nil
}

func awaitEchoReply(session *slim.Session, payload []byte, timeout time.Duration) error {
	reply, err := awaitMatchingMessage(session, timeout, func(current []byte) bool {
		return bytes.Equal(current, buildEchoReplyPayload(payload))
	})
	if err != nil {
		return err
	}
	if !bytes.Equal(reply, buildEchoReplyPayload(payload)) {
		return fmt.Errorf("unexpected echo reply payload")
	}
	return nil
}

func confirmPubDelivery(session *slim.Session, clientID int, expectedMessages int64) error {
	drainRequest := []byte(fmt.Sprintf("%s%d", string(drainPayload), clientID))
	if err := session.PublishAndWait(drainRequest, nil, nil); err != nil {
		return fmt.Errorf("publish drain request: %w", err)
	}

	reply, err := awaitMatchingMessage(session, 5*time.Second, func(payload []byte) bool {
		if !bytes.HasPrefix(payload, drainAckPrefix) {
			return false
		}
		return strings.Contains(string(payload), fmt.Sprintf("client=%d", clientID))
	})
	if err != nil {
		return fmt.Errorf("await drain reply: %w", err)
	}

	sessionReceived, err := parseIntField(reply, "session_received")
	if err != nil {
		return fmt.Errorf("parse drain reply: %w", err)
	}
	if sessionReceived <= 0 {
		return fmt.Errorf("sink reported zero delivered messages")
	}
	if sessionReceived < expectedMessages {
		return fmt.Errorf("sink reported %d delivered messages, expected at least %d", sessionReceived, expectedMessages)
	}

	return nil
}

func awaitMatchingMessage(session *slim.Session, timeout time.Duration, matcher func([]byte) bool) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, fmt.Errorf("timed out waiting for expected message")
		}

		msg, err := session.GetMessage(&remaining)
		if err != nil {
			return nil, err
		}
		if matcher(msg.Payload) {
			return append([]byte{}, msg.Payload...), nil
		}
		log.Printf("ignoring unexpected session message while waiting for acknowledgement: len=%d", len(msg.Payload))
	}
}

func buildEchoReplyPayload(payload []byte) []byte {
	return append(append([]byte{}, echoAckPrefix...), payload...)
}

func retireOldestCompletion(inFlight *[]*slim.CompletionHandle, clientID int) error {
	if len(*inFlight) == 0 {
		return nil
	}
	if len(*inFlight) < cap(*inFlight) {
		return nil
	}

	handle := (*inFlight)[0]
	remaining := (*inFlight)[1:]
	*inFlight = remaining
	if handle == nil {
		return nil
	}
	if err := handle.WaitAsync(); err != nil {
		return fmt.Errorf("client %d async publish completion failed: %w", clientID, err)
	}
	return nil
}

func waitForTurn(start time.Time, sent int64, clientRate float64) {
	for {
		elapsed := time.Since(start)
		allowedMessages := int64(elapsed.Seconds() * clientRate)
		if allowedMessages > sent {
			return
		}

		nextMessageAt := time.Duration(float64(sent+1) / clientRate * float64(time.Second))
		wait := nextMessageAt - elapsed
		if wait <= 0 {
			return
		}
		if wait > time.Millisecond {
			time.Sleep(time.Millisecond)
		} else {
			time.Sleep(wait)
		}
	}
}

func aggregate(results []runStats, duration time.Duration, cfg config) aggregateStats {
	var totalMessages int64
	var totalBytes int64
	var errorCount int64
	allLatencies := make([]time.Duration, 0, len(results)*1024)

	for _, result := range results {
		totalMessages += result.Sent
		totalBytes += result.Bytes
		errorCount += result.Errors
		allLatencies = append(allLatencies, result.Latencies...)
	}

	agg := aggregateStats{
		TotalMessages: totalMessages,
		TotalBytes:    totalBytes,
		Duration:      duration,
		MPS:           float64(totalMessages) / duration.Seconds(),
		MBPS:          (float64(totalBytes) / 1024 / 1024) / duration.Seconds(),
		ErrorCount:    errorCount,
		Config:        cfg,
	}

	if len(allLatencies) == 0 {
		return agg
	}

	sort.Slice(allLatencies, func(i, j int) bool { return allLatencies[i] < allLatencies[j] })
	var sum time.Duration
	for _, latency := range allLatencies {
		sum += latency
	}

	count := len(allLatencies)
	agg.Mean = sum / time.Duration(count)
	agg.Min = allLatencies[0]
	agg.Max = allLatencies[count-1]
	agg.P50 = percentile(allLatencies, 0.50)
	agg.P90 = percentile(allLatencies, 0.90)
	agg.P99 = percentile(allLatencies, 0.99)

	avg := float64(agg.Mean)
	var variance float64
	for _, latency := range allLatencies {
		delta := float64(latency) - avg
		variance += delta * delta
	}
	agg.StdDev = time.Duration(math.Sqrt(variance / float64(count)))

	return agg
}

func percentile(values []time.Duration, p float64) time.Duration {
	index := int(math.Ceil(float64(len(values))*p)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func printConsoleReport(agg aggregateStats) {
	fmt.Println("\nRate Client Stats:")
	fmt.Printf("  Throughput: %.0f msgs/sec ~ %.2f MB/sec\n", agg.MPS, agg.MBPS)
	fmt.Printf("  Latencies:  [Min: %v | Mean: %v | Max: %v]\n", agg.Min, agg.Mean, agg.Max)
	fmt.Printf("  Errors:     %d\n", agg.ErrorCount)
	fmt.Println("----------------------------------------------------------------")
}

func writeMarkdownReport(outputPath string, agg aggregateStats) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer file.Close()

	tmpl, err := template.New("rate-client-report").Parse(reportTemplate)
	if err != nil {
		return fmt.Errorf("parse report template: %w", err)
	}

	if err := tmpl.Execute(file, agg); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	return nil
}

func localIDForClient(base string, id int, total int) string {
	if total == 1 {
		return base
	}
	return suffixIdentity(base, id)
}

func destIDForClient(base string, id int, total int, shard bool) string {
	if !shard || total == 1 {
		return base
	}
	return suffixIdentity(base, id)
}

func suffixIdentity(base string, id int) string {
	parts := strings.Split(base, "/")
	parts[len(parts)-1] = fmt.Sprintf("%s-%d", parts[len(parts)-1], id)
	return strings.Join(parts, "/")
}

func connectToServer(serverAddr string) (uint64, error) {
	config := slim.NewInsecureClientConfig(serverAddr)
	connID, err := slim.GetGlobalService().Connect(config)
	if err != nil {
		return 0, fmt.Errorf("connect failed: %w", err)
	}

	return connID, nil
}

func createSubscribedApp(localID, secret string, connID uint64) (*slim.App, error) {
	appName, err := nameFromString(localID)
	if err != nil {
		return nil, err
	}

	app, err := slim.GetGlobalService().CreateAppWithSecret(appName, secret)
	if err != nil {
		return nil, fmt.Errorf("create app failed: %w", err)
	}

	if err := app.Subscribe(app.Name(), &connID); err != nil {
		app.Destroy()
		return nil, fmt.Errorf("subscribe failed: %w", err)
	}

	return app, nil
}

func nameFromString(value string) (*slim.Name, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid name format: %s", value)
	}

	return slim.NewName(parts[0], parts[1], parts[2]), nil
}

func parseIntField(payload []byte, key string) (int64, error) {
	for _, field := range strings.Fields(string(payload)) {
		if strings.HasPrefix(field, key+"=") {
			return strconv.ParseInt(strings.TrimPrefix(field, key+"="), 10, 64)
		}
	}
	return 0, fmt.Errorf("missing %s field", key)
}
