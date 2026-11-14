package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"testing"
)

type responseTransport struct {
	response *http.Response
}

func (transport responseTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return transport.response, nil
}

type trackedResponseBody struct {
	io.Reader
	bytesRead int
	closed    bool
}

func (body *trackedResponseBody) Read(buffer []byte) (int, error) {
	count, err := body.Reader.Read(buffer)
	body.bytesRead += count
	return count, err
}

func (body *trackedResponseBody) Close() error {
	body.closed = true
	return nil
}

func TestGotifyClientSendResponseHandling(t *testing.T) {
	const responseLimit = 64 * 1024
	tests := []struct {
		name      string
		status    int
		body      string
		wantError string
		wantRead  int
	}{
		{name: "success closes body without decoding", status: http.StatusOK, body: "not JSON", wantRead: 8},
		{name: "bounded success", status: http.StatusOK, body: strings.Repeat("x", responseLimit+1), wantRead: responseLimit},
		{name: "empty error", status: http.StatusInternalServerError, wantError: "gotify server returned status: 500"},
		{name: "whitespace error", status: http.StatusBadGateway, body: " \n\t ", wantError: "gotify server returned status: 502", wantRead: 4},
		{name: "trimmed diagnostic", status: http.StatusUnauthorized, body: " \ninvalid token\t ", wantError: "gotify server returned status 401: invalid token", wantRead: 17},
		{name: "bounded diagnostic", status: http.StatusServiceUnavailable, body: strings.Repeat("x", responseLimit) + "must not be read", wantError: "gotify server returned status 503: " + strings.Repeat("x", responseLimit), wantRead: responseLimit},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := &trackedResponseBody{Reader: strings.NewReader(test.body)}
			client := &GotifyClient{
				URL:        "http://gotify.invalid",
				Token:      "test-token",
				HTTPClient: &http.Client{Transport: responseTransport{response: &http.Response{StatusCode: test.status, Body: body}}},
			}
			err := client.Send(context.Background(), GotifyMessage{Message: "test"})
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err == nil || err.Error() != test.wantError {
				t.Errorf("error mismatch: got length %d, want length %d", len(errorText(err)), len(test.wantError))
			}
			if body.bytesRead != test.wantRead {
				t.Errorf("read %d bytes, want %d", body.bytesRead, test.wantRead)
			}
			if !body.closed {
				t.Error("response body was not closed")
			}
		})
	}
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestGotifyClientSendReusesConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, `{"id":1,"message":"sent"}`)
	}))
	defer server.Close()
	client := &GotifyClient{URL: server.URL, Token: "test-token", HTTPClient: server.Client()}
	for index := 0; index < 2; index++ {
		reused := false
		ctx := httptrace.WithClientTrace(context.Background(), &httptrace.ClientTrace{
			GotConn: func(info httptrace.GotConnInfo) { reused = info.Reused },
		})
		if err := client.Send(ctx, GotifyMessage{Message: "test"}); err != nil {
			t.Fatal(err)
		}
		if index == 1 && !reused {
			t.Error("second send did not reuse the connection")
		}
	}
}
