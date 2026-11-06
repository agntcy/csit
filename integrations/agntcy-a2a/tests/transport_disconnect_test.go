// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/onsi/gomega"
)

// disconnectingProxy cuts only its own client connections, leaving the fixture
// and its in-memory task store alive. Requests during the cut never reach it.
type disconnectingProxy struct {
	server    *httptest.Server
	forwarder *httputil.ReverseProxy
	transport *http.Transport
	cut       atomic.Bool
	forwarded atomic.Int64
	dropped   atomic.Int64
}

func newDisconnectingProxy(target *url.URL) *disconnectingProxy {
	proxy := &disconnectingProxy{
		forwarder: httputil.NewSingleHostReverseProxy(target),
		transport: http.DefaultTransport.(*http.Transport).Clone(),
	}
	proxy.forwarder.Transport = proxy.transport
	proxy.server = httptest.NewServer(http.HandlerFunc(proxy.serveHTTP))
	return proxy
}

func (proxy *disconnectingProxy) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	if proxy.cut.Load() {
		proxy.dropped.Add(1)
		// net/http closes the connection without sending an HTTP error response.
		panic(http.ErrAbortHandler)
	}

	proxy.forwarded.Add(1)
	proxy.forwarder.ServeHTTP(writer, request)
}

func (proxy *disconnectingProxy) disconnect() {
	proxy.cut.Store(true)
	proxy.server.CloseClientConnections()
}

func (proxy *disconnectingProxy) restore() {
	proxy.cut.Store(false)
}

func (proxy *disconnectingProxy) close() {
	proxy.server.Close()
	proxy.transport.CloseIdleConnections()
}

func TestDisconnectingProxy(t *testing.T) {
	check := gomega.NewWithT(t)
	var backendRequests atomic.Int64
	backend := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		backendRequests.Add(1)
		_, _ = io.WriteString(writer, request.URL.Path)
	}))
	t.Cleanup(backend.Close)

	target, err := url.Parse(backend.URL)
	check.Expect(err).NotTo(gomega.HaveOccurred())
	proxy := newDisconnectingProxy(target)
	t.Cleanup(proxy.close)

	transport := http.DefaultTransport.(*http.Transport).Clone()
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second}
	requestURL := proxy.server.URL + "/task/read"

	response, err := client.Post(requestURL, "application/json", http.NoBody)
	check.Expect(err).NotTo(gomega.HaveOccurred())
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	check.Expect(err).NotTo(gomega.HaveOccurred())
	check.Expect(string(body)).To(gomega.Equal("/task/read"))
	check.Expect(backendRequests.Load()).To(gomega.Equal(int64(1)))

	proxy.disconnect()
	transport.CloseIdleConnections()
	response, err = client.Post(requestURL, "application/json", http.NoBody)
	check.Expect(err).To(gomega.HaveOccurred())
	check.Expect(response).To(gomega.BeNil())
	check.Expect(proxy.dropped.Load()).To(gomega.BeNumerically(">", 0))
	check.Expect(backendRequests.Load()).To(gomega.Equal(int64(1)))

	proxy.restore()
	response, err = client.Post(requestURL, "application/json", http.NoBody)
	check.Expect(err).NotTo(gomega.HaveOccurred())
	body, err = io.ReadAll(response.Body)
	response.Body.Close()
	check.Expect(err).NotTo(gomega.HaveOccurred())
	check.Expect(string(body)).To(gomega.Equal("/task/read"))
	check.Expect(backendRequests.Load()).To(gomega.Equal(int64(2)))
}
