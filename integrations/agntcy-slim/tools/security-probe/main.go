// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net"
	"sync"
	"time"
)

// security-probe: A tool to attempt invalid connections, test security controls, and stress test
// Usage: security-probe --mode flood --target localhost:46357 --connections 1000
func main() {
	mode := flag.String("mode", "bad-tls", "Attack mode: bad-tls | mitm-tamper | flood | slow-handshake")
	target := flag.String("target", "localhost:46357", "Target SLIM Data Plane Address")
	connections := flag.Int("connections", 100, "Number of concurrent connections for flood/slow modes")

	flag.Parse()

	log.Printf("Starting security-probe in mode: %s against %s", *mode, *target)

	switch *mode {
	case "bad-tls":
		runBadTLS(*target)
	case "mitm-tamper":
		runMitmTamper(*target)
	case "flood":
		runConnectionFlood(*target, *connections)
	case "slow-handshake":
		runSlowHandshake(*target, *connections)
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

func runBadTLS(target string) {
	// Attempt connection with skip verify and empty certs
	conf := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{}, // Empty certs
	}

	conn, err := tls.Dial("tcp", target, conf)
	if err != nil {
		log.Printf("[PASS] Connection rejected as expected: %v", err)
		return
	}
	defer conn.Close()

	// If we connected, it's a failure (unless checking specific handshake phase)
	log.Printf("[FAIL] Connection established with invalid credentials!")

	// Attempt to send garbage
	_, err = conn.Write([]byte("GARBAGE_DATA"))
	if err != nil {
		log.Printf("[PASS] Connection closed on write: %v", err)
	} else {
		log.Printf("[FAIL] Start Sending Garbage... (this should close soon)")
	}
}

func runMitmTamper(target string) {
	// setup a local proxy that intercepts traffic
	localListener, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Fatalf("Failed to start proxy: %v", err)
	}
	defer localListener.Close()

	log.Printf("MITM Proxy listening on :9999 forwarding to %s", target)

	for {
		client, err := localListener.Accept()
		if err != nil {
			continue
		}
		go handleProxy(client, target)
	}
}

func handleProxy(client net.Conn, target string) {
	server, err := net.Dial("tcp", target)
	if err != nil {
		client.Close()
		return
	}
	defer server.Close()
	defer client.Close()

	// Simplified pipe with tampering logic placeholder
	go pipe(client, server, true) // Client -> Server (Tamper)
	pipe(server, client, false)   // Server -> Client
}

func pipe(src, dst net.Conn, tamper bool) {
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if err != nil {
			return
		}
		data := buf[:n]

		if tamper {
			// Simple tamper: flip a byte
			if len(data) > 10 {
				data[10] ^= 0xFF
				log.Println("Tampered with packet!")
			}
		}

		dst.Write(data)
	}
}

func runConnectionFlood(target string, count int) {
	var wg sync.WaitGroup
	log.Printf("Starting flood of %d connections...", count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn, err := net.Dial("tcp", target)
			if err != nil {
				// Expected if server is overwhelmed or rejects
				return
			}
			// Hold connection open
			defer conn.Close()
			time.Sleep(30 * time.Second)
		}(i)

		if i%100 == 0 {
			time.Sleep(10 * time.Millisecond) // Throttle slightly to avoid self-DoS
		}
	}
	wg.Wait()
	log.Println("Flood complete")
}

func runSlowHandshake(target string, count int) {
	var wg sync.WaitGroup
	log.Printf("Starting slow handshake attack with %d connections...", count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn, err := net.Dial("tcp", target)
			if err != nil {
				return
			}
			defer conn.Close()

			// Send partial ClientHello (or just garbage start) then stall
			// A real ClientHello byte sequence would be better here for TLS stalling
			// For now, we simulate a "connect and hang" which is similar to slowloris at TCP level
			// To be more precise, we would write the first few bytes of a TLS handshake
			conn.Write([]byte{0x16, 0x03, 0x01}) // TLS Record Layer content type Handshake, Version 3.1

			time.Sleep(30 * time.Second) // Stall
		}(i)
	}
	wg.Wait()
	log.Println("Slow handshake attack complete")
}
