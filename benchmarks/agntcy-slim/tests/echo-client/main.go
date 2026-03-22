// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
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

type counterSet struct {
	receivedMessages atomic.Int64
	receivedBytes    atomic.Int64
	replyMessages    atomic.Int64
	errors           atomic.Int64
	warmupMessages   atomic.Int64
	warmupReplies    atomic.Int64
	drainMessages    atomic.Int64
	drainReplies     atomic.Int64
	firstMessageUnix atomic.Int64
	lastMessageUnix  atomic.Int64
}

func main() {
	local := flag.String("local", "agntcy/demo/echo", "Local ID in org/namespace/app format")
	clients := flag.Int("clients", 1, "Number of receiver identities to keep active")
	mode := flag.String("mode", "echo", "Responder mode: echo, sink, or blackhole")
	statsFile := flag.String("stats-file", "", "Path to write responder stats")
	server := flag.String("server", "http://127.0.0.1:46357", "SLIM server endpoint")
	secret := flag.String("shared-secret", defaultSharedSecret, "Shared secret")
	flag.Parse()

	if *clients < 1 {
		log.Fatal("clients must be >= 1")
	}
	if *mode != "echo" && *mode != "sink" && *mode != "blackhole" {
		log.Fatal("mode must be echo, sink, or blackhole")
	}

	slim.InitializeWithDefaults()
	connID, err := connectToServer(*server)
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}
	counters := &counterSet{}
	started := time.Now()
	stopStats := make(chan struct{})
	if *statsFile != "" {
		go writeStatsLoop(*statsFile, *mode, counters, started, stopStats)
		defer close(stopStats)
	}
	apps := make([]*slim.App, 0, *clients+1)
	for _, identity := range receiverIdentities(*local, *clients) {
		app, err := createSubscribedApp(identity, *secret, connID)
		if err != nil {
			log.Fatalf("failed to create app %s: %v", identity, err)
		}
		apps = append(apps, app)
		fmt.Printf("ready local=%s conn_id=%d app_id=%d\n", identity, connID, app.Id())
	}

	var wg sync.WaitGroup
	for _, app := range apps {
		wg.Add(1)
		go func(currentApp *slim.App) {
			defer wg.Done()
			serveApp(currentApp, *mode, counters)
		}(app)
	}

	wg.Wait()
}

func serveApp(app *slim.App, mode string, counters *counterSet) {
	fmt.Printf("listening app_id=%d\n", app.Id())
	for {
		session, err := app.ListenForSession(nil)
		if err != nil {
			log.Printf("listen retry: %v", err)
			continue
		}
		log.Printf("accepted session app_id=%d mode=%s", app.Id(), mode)

		go handleSession(app, session, mode, counters)
	}
}

func handleSession(app *slim.App, session *slim.Session, mode string, counters *counterSet) {
	var sessionReceived int64
	var sessionBytes int64

	defer func() {
		if err := app.DeleteSessionAndWaitAsync(session); err != nil {
			log.Printf("session cleanup failed: %v", err)
		}
	}()

	for {
		timeout := 30 * time.Second
		msg, err := session.GetMessage(&timeout)
		if err != nil {
			log.Printf("session ended: %v", err)
			return
		}

		if bytes.Equal(msg.Payload, warmupPayload) {
			log.Printf("received warmup app_id=%d mode=%s", app.Id(), mode)
			counters.warmupMessages.Add(1)
			if err := session.PublishToAndWait(msg.Context, append([]byte{}, warmupAckPrefix...), nil, nil); err != nil {
				log.Printf("warmup reply failed: %v", err)
				counters.errors.Add(1)
				return
			}
			counters.warmupReplies.Add(1)
			continue
		}

		if bytes.HasPrefix(msg.Payload, drainPayload) {
			log.Printf("received drain request app_id=%d mode=%s", app.Id(), mode)
			counters.drainMessages.Add(1)
			ackPayload := []byte(fmt.Sprintf(
				"%sclient=%s session_received=%d session_bytes=%d global_received=%d",
				string(drainAckPrefix),
				string(bytes.TrimPrefix(msg.Payload, drainPayload)),
				sessionReceived,
				sessionBytes,
				counters.receivedMessages.Load(),
			))
			if err := session.PublishToAndWait(msg.Context, ackPayload, nil, nil); err != nil {
				log.Printf("drain reply failed: %v", err)
				counters.errors.Add(1)
				return
			}
			counters.drainReplies.Add(1)
			continue
		}

		if mode != "blackhole" {
			counters.receivedMessages.Add(1)
			counters.receivedBytes.Add(int64(len(msg.Payload)))
			nowUnix := time.Now().UnixNano()
			if counters.firstMessageUnix.Load() == 0 {
				counters.firstMessageUnix.CompareAndSwap(0, nowUnix)
			}
			counters.lastMessageUnix.Store(nowUnix)
			sessionReceived++
			sessionBytes += int64(len(msg.Payload))
		}

		if mode == "echo" {
			replyPayload := append(append([]byte{}, echoAckPrefix...), msg.Payload...)
			if err := session.PublishToAndWait(msg.Context, replyPayload, nil, nil); err != nil {
				log.Printf("reply failed: %v", err)
				counters.errors.Add(1)
				return
			}
			counters.replyMessages.Add(1)
		}
	}
}

func writeStatsLoop(statsFile string, mode string, counters *counterSet, started time.Time, stop <-chan struct{}) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	writeSnapshot := func() {
		elapsed := time.Since(started)
		seconds := elapsed.Seconds()
		if seconds <= 0 {
			seconds = 1e-9
		}
		receivedMessages := counters.receivedMessages.Load()
		receivedBytes := counters.receivedBytes.Load()
		replyMessages := counters.replyMessages.Load()
		errors := counters.errors.Load()
		warmupMessages := counters.warmupMessages.Load()
		warmupReplies := counters.warmupReplies.Load()
		drainMessages := counters.drainMessages.Load()
		drainReplies := counters.drainReplies.Load()
		activeSeconds := seconds
		firstMessageUnix := counters.firstMessageUnix.Load()
		lastMessageUnix := counters.lastMessageUnix.Load()
		if receivedMessages > 1 && firstMessageUnix > 0 && lastMessageUnix > firstMessageUnix {
			activeSeconds = float64(lastMessageUnix-firstMessageUnix) / float64(time.Second)
			if activeSeconds <= 0 {
				activeSeconds = 1e-9
			}
		}
		content := fmt.Sprintf(
			"mode=%s\nreceived_messages=%d\nreceived_bytes=%d\nreply_messages=%d\nerrors=%d\nwarmup_messages=%d\nwarmup_replies=%d\ndrain_messages=%d\ndrain_replies=%d\nelapsed_seconds=%.6f\nactive_receive_seconds=%.6f\nreceive_mps=%.2f\nreceive_mbps=%.2f\nactive_receive_mps=%.2f\nactive_receive_mbps=%.2f\n",
			mode,
			receivedMessages,
			receivedBytes,
			replyMessages,
			errors,
			warmupMessages,
			warmupReplies,
			drainMessages,
			drainReplies,
			seconds,
			activeSeconds,
			float64(receivedMessages)/seconds,
			(float64(receivedBytes)/1024/1024)/seconds,
			float64(receivedMessages)/activeSeconds,
			(float64(receivedBytes)/1024/1024)/activeSeconds,
		)
		if err := os.WriteFile(statsFile, []byte(content), 0644); err != nil {
			log.Printf("write stats failed: %v", err)
		}
	}

	writeSnapshot()
	for {
		select {
		case <-stop:
			writeSnapshot()
			return
		case <-ticker.C:
			writeSnapshot()
		}
	}
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

func receiverIdentities(base string, clients int) []string {
	if clients == 1 {
		return []string{base}
	}

	identities := make([]string, 0, clients+1)
	identities = append(identities, base)
	for index := 1; index <= clients; index++ {
		identities = append(identities, suffixIdentity(base, index))
	}
	return identities
}

func suffixIdentity(base string, id int) string {
	parts := strings.Split(base, "/")
	parts[len(parts)-1] = fmt.Sprintf("%s-%d", parts[len(parts)-1], id)
	return strings.Join(parts, "/")
}
