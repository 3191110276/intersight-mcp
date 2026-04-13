package testutil

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type TCP4Server struct {
	URL      string
	server   *http.Server
	listener net.Listener
	client   *http.Client
}

type TCP4TLSServer struct {
	URL    string
	server *httptest.Server
}

func NewTCP4Server(t *testing.T, handler http.Handler) *TCP4Server {
	t.Helper()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		if shouldSkipBindError(err) {
			t.Skipf("skipping listener-based test in restricted environment: %v", err)
		}
		t.Fatalf("net.Listen() error = %v", err)
	}

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ts := &TCP4Server{
		URL:      fmt.Sprintf("http://%s", listener.Addr().String()),
		server:   srv,
		listener: listener,
		client:   &http.Client{},
	}

	go func() {
		_ = srv.Serve(listener)
	}()

	return ts
}

func NewTCP4TLSServer(t *testing.T, handler http.Handler) *TCP4TLSServer {
	t.Helper()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		if shouldSkipBindError(err) {
			t.Skipf("skipping listener-based test in restricted environment: %v", err)
		}
		t.Fatalf("net.Listen() error = %v", err)
	}

	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.EnableHTTP2 = false
	server.StartTLS()
	return &TCP4TLSServer{
		URL:    server.URL,
		server: server,
	}
}

func (s *TCP4Server) Client() *http.Client {
	return s.client
}

func (s *TCP4TLSServer) Client() *http.Client {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Client()
}

func (s *TCP4Server) Close() {
	if s == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = s.server.Shutdown(ctx)
}

func (s *TCP4TLSServer) Close() {
	if s == nil || s.server == nil {
		return
	}
	s.server.CloseClientConnections()
	s.server.Close()
}

func shouldSkipBindError(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var sysErr *os.SyscallError
		if errors.As(opErr.Err, &sysErr) && strings.Contains(strings.ToLower(sysErr.Err.Error()), "operation not permitted") {
			return true
		}
	}
	return strings.Contains(strings.ToLower(err.Error()), "operation not permitted")
}
