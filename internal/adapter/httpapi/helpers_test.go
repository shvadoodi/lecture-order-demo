package httpapi

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestDecodeBodyLimitsAndIOFailure(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    io.Reader
		wantErr bool
	}{
		{"exact limit", strings.NewReader("{}" + strings.Repeat(" ", (1<<20)-2)), false},
		{"over limit", strings.NewReader("{}" + strings.Repeat(" ", (1<<20)-1)), true},
		{"large object", strings.NewReader("{\"customerId\":\"" + strings.Repeat("x", 1<<20) + "\"}"), true},
		{"reader failure", failingReader{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/orders", tc.body)
			var dst map[string]any
			err := decodeJSONBody(request, &dst)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestWriteJSONLogsEncodingFailure(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previous)
	response := httptest.NewRecorder()
	writeJSON(response, http.StatusOK, make(chan int))
	if !strings.Contains(output.String(), "write response error") {
		t.Fatal("encoding failure not logged")
	}
}

func TestLoggingMiddleware(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previous)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(http.StatusAccepted) })
	response := httptest.NewRecorder()
	loggingMiddleware(next).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/example", nil))
	if !called || response.Code != http.StatusAccepted || !strings.Contains(output.String(), "GET /example") {
		t.Fatal("middleware did not log and delegate")
	}
}
