package main

import (
	"bufio"
	"net"
	"net/http"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	Code        int
	Size        int64
	wroteHeader bool
}

var (
	_ http.ResponseWriter = (*responseWriterWrapper)(nil)
	_ http.Hijacker       = (*responseWriterWrapper)(nil)
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

func (w *responseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	controller := http.NewResponseController(w.ResponseWriter)
	return controller.Hijack()
}
