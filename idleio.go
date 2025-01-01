package idleio

import (
	"fmt"
	"io"
	"time"
)

type Reader struct {
	rc          readController
	r           io.Reader
	idleTimeout time.Duration
}

func NewReader(rc readController, r io.Reader, idleTimeout time.Duration) *Reader {
	return &Reader{
		rc:          rc,
		r:           r,
		idleTimeout: idleTimeout,
	}
}

func (r *Reader) Read(p []byte) (int, error) {
	if err := r.rc.SetReadDeadline(time.Now().Add(r.idleTimeout)); err != nil {
		return 0, fmt.Errorf("idleio: %w", err)
	}
	return r.r.Read(p)
}

type Writer struct {
	wc          writeController
	w           io.Writer
	idleTimeout time.Duration
}

func NewWriter(wc writeController, w io.Writer, idleTimeout time.Duration) *Writer {
	return &Writer{
		wc:          wc,
		w:           w,
		idleTimeout: idleTimeout,
	}
}

func (w *Writer) Write(p []byte) (int, error) {
	if err := w.wc.SetWriteDeadline(time.Now().Add(w.idleTimeout)); err != nil {
		return 0, fmt.Errorf("idleio: %w", err)
	}
	return w.w.Write(p)
}

type readController interface {
	SetReadDeadline(deadline time.Time) error
}

type writeController interface {
	SetWriteDeadline(deadline time.Time) error
}
