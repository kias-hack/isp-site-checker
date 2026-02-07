package checker

import (
	"context"
	"net"
	"net/http"
	"time"
)

func createClient(host string, port string) *http.Client {
	transport := &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{
				Timeout: 10 * time.Second,
				// KeepAlive: 10 * time.Second,
			}).DialContext(ctx, network, net.JoinHostPort(host, port))
		},
	}

	// TODO add configurable timeout and keepalive

	return &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}
}
