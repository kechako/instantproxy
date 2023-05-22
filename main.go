package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"time"

	flag "github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
)

var (
	addr string
	cert string
	key  string
)

func init() {
	flag.StringVar(&addr, "http", ":8080", "IP address and port number to bind.")
	flag.StringVar(&cert, "cert", "", "TLS certificate file")
	flag.StringVar(&key, "key", "", "TLS private key file")
}

func printError(err error, exit bool) {
	fmt.Fprintf(os.Stderr, "[ERROR] %+v\n", err)
	if exit {
		os.Exit(1)
	}
}

var accessLogger = log.New(os.Stdout, "", log.LstdFlags)

func accessLog(code int, size int64, method, path string) {
	accessLogger.Printf("[%d]: %s %s (%d bytes)", code, method, path, size)
}

func accessLogHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapper := responseWriterWrapper{ResponseWriter: w}

		next.ServeHTTP(&wrapper, r)

		accessLog(wrapper.Code, wrapper.Size, r.Method, r.URL.Path)
	})
}

func reverseProxy(target *url.URL) http.Handler {
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.SetXForwarded()
		},
	}
}

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		printError(errors.New("proxy target is not specified"), true)
	}
	target, err := url.Parse(flag.Arg(0))
	if err != nil {
		printError(fmt.Errorf("failed to parse target URL: %w", err), true)
		return
	}

	server := &http.Server{
		Addr:    addr,
		Handler: accessLogHandler(reverseProxy(target)),
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		<-ctx.Done()

		toCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return server.Shutdown(toCtx)
	})

	group.Go(func() error {
		if cert != "" && key != "" {
			return server.ListenAndServeTLS(cert, key)
		} else {
			return server.ListenAndServe()
		}
	})

	fmt.Printf("Start server [%s], and forwared to [%s]\n", addr, target)

	err = group.Wait()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		printError(err, true)
	}
}
