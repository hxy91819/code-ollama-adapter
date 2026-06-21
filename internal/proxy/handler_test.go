package proxy

import (
	"net/http"
	"strings"
	"testing"
)

type flushRecorder struct {
	header  http.Header
	body    strings.Builder
	flushes int
	status  int
}

func (r *flushRecorder) Header() http.Header {
	if r.header == nil {
		r.header = http.Header{}
	}
	return r.header
}

func (r *flushRecorder) Write(body []byte) (int, error) {
	return r.body.Write(body)
}

func (r *flushRecorder) WriteHeader(status int) {
	r.status = status
}

func (r *flushRecorder) Flush() {
	r.flushes++
}

func TestCopyResponseBodyFlushesWritableChunks(t *testing.T) {
	recorder := &flushRecorder{}

	if err := copyResponseBody(recorder, strings.NewReader("stream chunk")); err != nil {
		t.Fatal(err)
	}

	if recorder.body.String() != "stream chunk" {
		t.Fatalf("body = %q", recorder.body.String())
	}
	if recorder.flushes == 0 {
		t.Fatal("expected at least one flush for streaming response writer")
	}
}
