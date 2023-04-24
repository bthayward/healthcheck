package healthcheck

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_httpPingUP(t *testing.T) {

	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result := httpPing(ctx, server.Client(), Endpoint{
		Name: "test server",
		Url:  server.URL,
	})
	if result.err != nil || result.status != UP {
		t.Errorf("Unexpected result %v %v", result.err, result.status)
	}
}

func Test_httpPingErrorCodeDOWN(t *testing.T) {

	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result := httpPing(ctx, server.Client(), Endpoint{
		Name: "test server",
		Url:  server.URL,
	})
	if result.err != nil || result.status != DOWN {
		t.Errorf("Unexpected result %v %v", result.err, result.status)
	}
}

func Test_httpPingTimeoutDOWN(t *testing.T) {

	ctx := context.Background()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Errorf("Failed to start listener %v", err)
	}
	defer listener.Close()

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		}),
	}
	defer server.Shutdown(ctx)
	go func() { // TODO Race condition, add wait point
		_ = server.Serve(listener)
	}()

	client := &http.Client{
		Timeout: 500 * time.Millisecond,
	}

	result := httpPing(ctx, client, Endpoint{
		Name: "test server",
		Url:  fmt.Sprintf("http://localhost:%v/", listener.Addr().(*net.TCPAddr).Port),
	})
	if result.err != nil || result.status != DOWN {
		t.Errorf("Unexpected result %v %v", result.err, result.status)
	}
}

func Test_httpPingErrorDOWN(t *testing.T) {

	ctx := context.Background()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Errorf("Failed to start listener %v", err)
	}
	defer listener.Close()

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}
	defer server.Shutdown(ctx)
	go func() { // TODO Race condition, add wait point
		_ = server.Serve(listener)
	}()

	client := &http.Client{
		Timeout: 500 * time.Millisecond,
	}

	result := httpPing(ctx, client, Endpoint{
		Name: "test server",
		Url:  fmt.Sprintf("fake://localhost:%v/", listener.Addr().(*net.TCPAddr).Port), // FAKE PROTOCOL
	})
	if result.err == nil || result.status != UNDEFINED {
		t.Errorf("Unexpected result %v", result.status)
	}
}
