package client

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/http2"
)

// Config of the http2 client
type Config struct {
	Addr     string
	Method   string
	Insecure bool
	Headers  []string
	Data     string
}

// Client is a http2 client
type Client struct {
	Config
	client http.Client
}

// New returns a new client
func New(cfg Config) (*Client, error) {
	// Parse url
	url, err := url.Parse(cfg.Addr)
	if err != nil {
		return nil, err
	}

	var tlsConfig *tls.Config
	tlsEnabled := url.Scheme == "https"
	if tlsEnabled {
		tlsConfig = &tls.Config{InsecureSkipVerify: cfg.Insecure}
	}

	// Create http2 client
	client := http.Client{
		Transport: &http2.Transport{
			AllowHTTP: !tlsEnabled,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				if tlsEnabled {
					return tls.Dial(network, addr, cfg)
				}
				return net.Dial(network, addr)
			},
			TLSClientConfig: tlsConfig,
		},
	}

	return &Client{
		client: client,
		Config: cfg,
	}, nil
}

// Run launches a client
func (c *Client) Run() error {
	// Set body reader
	var body io.Reader
	if c.Data != "" {
		body = io.MultiReader(strings.NewReader(c.Data), os.Stdin)
	} else {
		body = os.Stdin
	}

	// Establish http connection
	req, err := http.NewRequest(c.Method, c.Addr, body)
	if c.Headers != nil {
		for _, h := range c.Headers {
			kv := strings.Split(h, ":")
			if len(kv) != 2 {
				return errors.New("client: error parsing header")
			}
			k := strings.Trim(kv[0], " ")
			v := strings.Trim(kv[1], " ")
			req.Header.Add(k, v)
		}
	}
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println(err)
		}
	}()

	log.Println(fmt.Sprintf("Connected to %s", c.Addr))
	io.Copy(os.Stdout, resp.Body)
	return nil
}
