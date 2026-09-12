// Package mt5bridge implements the read-only contract used to inspect an XM
// Demo terminal. It intentionally exposes no order or account mutation methods.
package mt5bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const APIVersion = "v1"

type Health struct {
	Status   string `json:"status"`
	Version  string `json:"version"`
	Terminal string `json:"terminal"`
}
type Account struct {
	Login      int64   `json:"login"`
	Server     string  `json:"server"`
	Currency   string  `json:"currency"`
	Balance    float64 `json:"balance"`
	Equity     float64 `json:"equity"`
	Margin     float64 `json:"margin"`
	FreeMargin float64 `json:"free_margin"`
}
type Quote struct {
	Symbol   string `json:"symbol"`
	Bid, Ask float64
	AsOf     time.Time `json:"as_of"`
}
type Position struct {
	Ticket                                                        int64 `json:"ticket"`
	Symbol, Side                                                  string
	Volume, OpenPrice, CurrentPrice, StopLoss, TakeProfit, Profit float64
}

// Reader is the additive, broker-neutral read surface needed by LongTrade.
type Reader interface {
	Health(context.Context) (*Health, error)
	Account(context.Context) (*Account, error)
	Quote(context.Context, string) (*Quote, error)
	Positions(context.Context) ([]Position, error)
}

type Client struct {
	base *url.URL
	http *http.Client
}

func New(rawURL string, timeout time.Duration) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(rawURL, "/"))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("mt5 bridge: invalid URL")
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Client{base: u, http: &http.Client{Timeout: timeout}}, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	relative, err := url.Parse("/api/" + APIVersion + path)
	if err != nil {
		return err
	}
	u := c.base.ResolveReference(relative)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("mt5 bridge unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mt5 bridge: %s returned %s", path, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("mt5 bridge: decode %s: %w", path, err)
	}
	return nil
}
func (c *Client) Health(ctx context.Context) (*Health, error) {
	var v Health
	return &v, c.get(ctx, "/health", &v)
}
func (c *Client) Account(ctx context.Context) (*Account, error) {
	var v Account
	return &v, c.get(ctx, "/account", &v)
}
func (c *Client) Quote(ctx context.Context, symbol string) (*Quote, error) {
	var v Quote
	return &v, c.get(ctx, "/quote?symbol="+url.QueryEscape(symbol), &v)
}
func (c *Client) Positions(ctx context.Context) ([]Position, error) {
	v := []Position{}
	return v, c.get(ctx, "/positions", &v)
}

var _ Reader = (*Client)(nil)
