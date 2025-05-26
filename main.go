package main

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/kechako/instantproxy/config"
	"github.com/kechako/instantproxy/proxy"
	"github.com/mattn/go-runewidth"
	flag "github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
)

var (
	addr    string
	cert    string
	key     string
	cfgName string
)

func init() {
	flag.StringVar(&addr, "http", ":8080", "IP address and port number to bind.")
	flag.StringVar(&cert, "cert", "", "TLS certificate file")
	flag.StringVar(&key, "key", "", "TLS private key file")
	flag.StringVar(&cfgName, "c", "config.toml", "Configuration file name")
}

func printError(err error, exit bool) {
	fmt.Fprintf(os.Stderr, "[ERROR] %+v\n", err)
	if exit {
		os.Exit(1)
	}
}

var accessLogger = log.New(os.Stdout, "", log.LstdFlags)

func accessLog(code int, size int64, method, scheme, host, path string) {
	accessLogger.Printf("[%d]: %s %s://%s%s (%d bytes)", code, method, scheme, host, path, size)
}

func accessLogHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapper := responseWriterWrapper{ResponseWriter: w}

		next.ServeHTTP(&wrapper, r)

		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		accessLog(wrapper.Code, wrapper.Size, r.Method, scheme, r.Host, r.URL.Path)
	})
}

func main() {
	flag.Parse()

	cfg, err := config.Load(cfgName)
	if err != nil {
		printError(err, true)
	}

	p, err := proxy.New(cfg)
	if err != nil {
		printError(err, true)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: accessLogHandler(p),
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

	fmt.Printf("Start server [%s], and forwared to:\n", addr)
	printProxyMap(p.ProxyMap())

	err = group.Wait()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		printError(err, true)
	}
}

func printProxyMap(i iter.Seq2[string, *url.URL]) {
	maxWidth := 0
	for host := range i {
		w := runewidth.StringWidth(host)
		if w > maxWidth {
			maxWidth = w
		}
	}

	for host, backendURL := range i {
		fmt.Printf("  %s => %s\n", runewidth.FillRight(host, maxWidth), backendURL)
	}
	fmt.Println()
}
