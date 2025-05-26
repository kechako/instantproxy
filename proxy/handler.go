package proxy

import (
	"fmt"
	"iter"
	"maps"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"
	"strings"

	"github.com/kechako/instantproxy/config"
)

type ReverseProxy struct {
	table map[string]*url.URL
	mux   *http.ServeMux
}

func New(cfg *config.Config) (*ReverseProxy, error) {
	table := map[string]*url.URL{}
	for _, server := range cfg.Servers {
		backendURL, err := url.Parse(server.BackendURL)
		if err != nil {
			return nil, fmt.Errorf("backend_url is not valid: %s: %w", server.BackendURL, err)
		}
		table[server.Host] = backendURL
	}

	mux := http.NewServeMux()

	for host, backendURL := range table {
		proxy := &httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				r.SetURL(backendURL)
				r.SetXForwarded()
			},
		}

		if host == "*" {
			host = ""
		}

		mux.Handle(host+"/", proxy)
	}

	return &ReverseProxy{
		table: table,
		mux:   mux,
	}, nil
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mux.ServeHTTP(w, r)
}

func (p *ReverseProxy) ProxyMap() iter.Seq2[string, *url.URL] {
	keys := slices.Collect(maps.Keys(p.table))
	slices.SortStableFunc(keys, func(a, b string) int {
		switch {
		case a == "*":
			return 1
		case b == "*":
			return -11
		default:
			return strings.Compare(a, b)
		}
	})

	return func(yield func(string, *url.URL) bool) {
		for _, host := range keys {
			backendURL := p.table[host]
			if !yield(host, backendURL) {
				return
			}
		}
	}
}
