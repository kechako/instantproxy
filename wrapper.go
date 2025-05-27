package main

import (
	"bufio"
	"net"
	"net/http"
	"time"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	controller  *http.ResponseController
	Code        int
	Size        int64
	wroteHeader bool
}

func newResponseWriterWrapper(w http.ResponseWriter) *responseWriterWrapper {
	return &responseWriterWrapper{
		ResponseWriter: w,
		controller:     http.NewResponseController(w),
		Code:           http.StatusOK, // Default to 200 OK
		Size:           0,
		wroteHeader:    false,
	}
}

var (
	_ http.ResponseWriter = (*responseWriterWrapper)(nil)
	_ http.Hijacker       = (*responseWriterWrapper)(nil)
	_ http.Flusher        = (*responseWriterWrapper)(nil)
)

func (w *responseWriterWrapper) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *responseWriterWrapper) Write(b []byte) (n int, err error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err = w.ResponseWriter.Write(b)
	if err == nil {
		w.Size += int64(n)
	}
	return
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		w.Code = statusCode
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) SetReadDeadline(deadline time.Time) error {
	return w.controller.SetReadDeadline(deadline)
}

func (w *responseWriterWrapper) SetWriteDeadline(deadline time.Time) error {
	return w.controller.SetWriteDeadline(deadline)
}

func (w *responseWriterWrapper) EnableFullDuplex() error {
	return w.controller.EnableFullDuplex()
}

func (w *responseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.controller.Hijack()
}

func (w *responseWriterWrapper) Flush() {
	err := w.controller.Flush()
	if err != nil {
		panic(err)
	}
}
