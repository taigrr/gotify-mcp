package main

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

type contextTransport func(*http.Request) (*http.Response, error)

func (transport contextTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestGotifyClientSendExpiredContext(t *testing.T) {
	for _, test := range []struct {
		name     string
		deadline bool
		want     error
	}{
		{name: "canceled", want: context.Canceled},
		{name: "deadline exceeded", deadline: true, want: context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			client, server := setupTestServer(t, func(http.ResponseWriter, *http.Request) {
				t.Error("expired request reached the server")
			})
			defer server.Close()
			ctx, cancel := context.WithCancel(context.Background())
			if test.deadline {
				cancel()
				ctx, cancel = context.WithDeadline(context.Background(), time.Unix(0, 0))
			} else {
				cancel()
			}
			defer cancel()
			if err := client.Send(ctx, GotifyMessage{Message: "test"}); !errors.Is(err, test.want) {
				t.Fatalf("Send error = %v, want wrapped %v", err, test.want)
			}
		})
	}
}

func TestGotifyClientSendInFlightCancellation(t *testing.T) {
	type contextKey struct{}
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "request metadata"))
	defer cancel()
	client := &GotifyClient{
		URL:   "http://gotify.invalid",
		Token: "test-token",
		HTTPClient: &http.Client{Transport: contextTransport(func(request *http.Request) (*http.Response, error) {
			if got := request.Context().Value(contextKey{}); got != "request metadata" {
				t.Errorf("context metadata = %v, want request metadata", got)
			}
			// Cancel only after the transport receives the request, without timing assumptions.
			cancel()
			select {
			case <-request.Context().Done():
				return nil, request.Context().Err()
			case <-time.After(time.Second):
				t.Error("request did not inherit caller cancellation")
				return nil, errors.New("request cancellation was not propagated")
			}
		})},
	}
	if err := client.Send(ctx, GotifyMessage{Message: "test"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Send error = %v, want wrapped context.Canceled", err)
	}
}
