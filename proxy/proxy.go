package proxy

import (
	"reposter/config"
	"net/http"
	"golang.org/x/net/proxy"
	"context"
	"net"
	"fmt"
)

func NewProxyTransport(conf *config.Config) *http.Transport {
	useAuth := true
	if conf.Proxy.User == "" || conf.Proxy.Password == "" {
		useAuth = false
	}

	var proxyAuth *proxy.Auth
	if useAuth {
		proxyAuth = &proxy.Auth{
			User:     conf.Proxy.User,
			Password: conf.Proxy.Password,
		}
	}

	tr := &http.Transport{
		DialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
			socksDialer, err := proxy.SOCKS5(
				"tcp",
				fmt.Sprintf("%s:%d",
					conf.Proxy.Host,
					conf.Proxy.Port,
				),
				proxyAuth,
				proxy.Direct,
			)
			if err != nil {
				return nil, err
			}

			return socksDialer.Dial(network, addr)
		},
	}

	return tr
}
