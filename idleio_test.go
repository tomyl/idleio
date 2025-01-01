package idleio_test

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tomyl/idleio"
)

func startHandler(t *testing.T, handler func(http.ResponseWriter, *http.Request)) string {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = l.Close()
	})

	go func() {
		_ = http.Serve(l, mux)
	}()

	return fmt.Sprintf("http://%s/", l.Addr())
}

func TestReadNoTimeout(t *testing.T) {
	url := startHandler(t, readHandler)

	resp, err := http.Post(url, "application/octet-stream", strings.NewReader("AAA"))
	require.NoError(t, err)

	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "3", string(buf))
}

func TestReadTimeout(t *testing.T) {
	url := startHandler(t, readHandler)

	reqBody := &slowReader{
		r:     strings.NewReader("AAA"),
		delay: 2 * time.Second,
	}

	resp, err := http.Post(url, "application/octet-stream", reqBody)
	require.NoError(t, err)

	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "timeout", string(buf))
}

func readHandler(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	rr := idleio.NewReader(rc, r.Body, time.Second)
	n, err := io.Copy(io.Discard, rr)
	if err != nil {
		var neterr net.Error
		if errors.As(err, &neterr) && neterr.Timeout() {
			fmt.Fprint(w, "timeout")
			return
		}

		fmt.Fprintf(w, "copy: %v\n", err)
		return
	}

	fmt.Fprintf(w, "%d", n)
}

func TestWriteNoTimeout(t *testing.T) {
	url := startHandler(t, writeHandler)

	resp, err := http.Get(url)
	require.NoError(t, err)

	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
}

func TestWriteTimeout(t *testing.T) {
	url := startHandler(t, writeHandler)

	resp, err := http.Get(url)
	require.NoError(t, err)

	defer resp.Body.Close()

	time.Sleep(2 * time.Second)

	_, err = io.ReadAll(resp.Body)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func writeHandler(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	ww := idleio.NewWriter(rc, w, time.Second)
	buf := strings.Repeat("A", 1_000_000)
	for range 10 {
		if _, err := io.WriteString(ww, buf); err != nil {
			return
		}
		if err := rc.Flush(); err != nil {
			return
		}
	}
}

type slowReader struct {
	r     io.Reader
	delay time.Duration
}

func (r *slowReader) Read(p []byte) (int, error) {
	time.Sleep(r.delay)
	return r.r.Read(p)
}
