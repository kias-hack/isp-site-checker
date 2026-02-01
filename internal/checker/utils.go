package checker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var readDir = os.ReadDir

func findSubdomain(owner string, domain string) []string {
	path := fmt.Sprintf("/var/www/%s/data/www/", owner)

	result := []string{domain}

	domain = "." + domain

	entries, err := readDir(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("unknown error while read dir %s: %w", path, err))
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == domain {
			continue
		}

		if strings.HasSuffix(entry.Name(), domain) {
			result = append(result, entry.Name())
		}
	}

	return result
}

func createClient(host string, port string) *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 10 * time.Second,
			}).DialContext(ctx, network, net.JoinHostPort(host, port))
		},
	}

	// TODO внедрить настройку timeout и keepalive

	return &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}
}
