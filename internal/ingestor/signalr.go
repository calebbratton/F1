// Package ingestor connects to the F1 Live Timing SignalR v2 stream
// (livetiming.formula1.com/signalr) without any API key.
// F1 exposes this feed publicly to power their own timing screen.
package ingestor

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

const (
	f1Host         = "livetiming.formula1.com"
	clientProtocol = "1.5"
	hub            = "Streaming"
)

// connectionData is sent as a JSON-encoded query param to identify which hub we want.
var connectionData = `[{"name":"` + hub + `"}]`

// Topics is the full list of feeds we subscribe to.
var Topics = []string{
	"Heartbeat",
	"SessionInfo",
	"SessionData",
	"DriverList",
	"TimingData",
	"TimingAppData",
	"TimingStats",
	"CarData.z",
	"Position.z",
	"WeatherData",
	"TrackStatus",
	"RaceControlMessages",
	"LapCount",
	"TeamRadio",
}

// negotiateResponse is returned by the /signalr/negotiate endpoint.
type negotiateResponse struct {
	ConnectionToken string `json:"ConnectionToken"`
}

// incomingMessage is the top-level SignalR v2 envelope.
type incomingMessage struct {
	// C is the stream cursor (used to resume after disconnect).
	C string `json:"C,omitempty"`
	// M carries the actual feed messages.
	M []feedMessage `json:"M,omitempty"`
	// S=1 signals the connection is initialized.
	S int `json:"S,omitempty"`
}

// feedMessage is one entry inside the M array.
// A = ["TopicName", <data>, <epochMs>]
type feedMessage struct {
	H string            `json:"H"` // hub name
	M string            `json:"M"` // method name ("feed")
	A []json.RawMessage `json:"A"` // arguments
}

// Client manages the SignalR v2 connection to F1 Live Timing.
type Client struct {
	conn    *websocket.Conn
	cursor  string
	onFeed  func(topic string, data json.RawMessage)
}

// NewClient creates a Client. onFeed is called for every incoming feed message.
func NewClient(onFeed func(topic string, data json.RawMessage)) *Client {
	return &Client{onFeed: onFeed}
}

// Connect negotiates, connects via WebSocket, and subscribes to all topics.
func (c *Client) Connect() error {
	token, err := negotiate()
	if err != nil {
		return fmt.Errorf("negotiate: %w", err)
	}

	wsURL := buildWSURL(token, c.cursor)
	dialer := websocket.DefaultDialer
	headers := http.Header{
		"User-Agent": {"Mozilla/5.0 (compatible; F1-API-Ingestor/1.0)"},
		"Origin":     {"https://www.formula1.com"},
	}

	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	c.conn = conn

	// Signal the start of the connection to the server.
	if err := start(token, c.cursor); err != nil {
		conn.Close()
		return fmt.Errorf("start: %w", err)
	}

	// Subscribe to all topics.
	if err := c.subscribe(); err != nil {
		conn.Close()
		return fmt.Errorf("subscribe: %w", err)
	}

	return nil
}

// ReadLoop blocks and dispatches messages until the connection closes.
func (c *Client) ReadLoop() error {
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return err
		}
		c.dispatch(raw)
	}
}

// Close closes the underlying WebSocket connection.
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) dispatch(raw []byte) {
	var msg incomingMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	// Save cursor for reconnect.
	if msg.C != "" {
		c.cursor = msg.C
	}

	for _, feed := range msg.M {
		if feed.M != "feed" || len(feed.A) < 2 {
			continue
		}
		var topic string
		if err := json.Unmarshal(feed.A[0], &topic); err != nil {
			continue
		}
		c.onFeed(topic, feed.A[1])
	}
}

func (c *Client) subscribe() error {
	msg := map[string]any{
		"H": hub,
		"M": "Subscribe",
		"A": []any{Topics},
		"I": 1,
	}
	return c.conn.WriteJSON(msg)
}

// negotiate calls the SignalR negotiate endpoint and returns the connection token.
func negotiate() (string, error) {
	u := &url.URL{
		Scheme: "https",
		Host:   f1Host,
		Path:   "/signalr/negotiate",
	}
	q := url.Values{}
	q.Set("clientProtocol", clientProtocol)
	q.Set("connectionData", connectionData)
	u.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; F1-API-Ingestor/1.0)")
	req.Header.Set("Referer", "https://www.formula1.com/")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var neg negotiateResponse
	if err := json.Unmarshal(body, &neg); err != nil {
		return "", fmt.Errorf("parse negotiate response: %w", err)
	}
	if neg.ConnectionToken == "" {
		return "", fmt.Errorf("empty connection token")
	}
	return neg.ConnectionToken, nil
}

// start sends the /signalr/start HTTP request which the server expects before streaming.
func start(token, cursor string) error {
	u := &url.URL{
		Scheme: "https",
		Host:   f1Host,
		Path:   "/signalr/start",
	}
	q := url.Values{}
	q.Set("transport", "webSockets")
	q.Set("clientProtocol", clientProtocol)
	q.Set("connectionToken", token)
	q.Set("connectionData", connectionData)
	if cursor != "" {
		q.Set("messageId", cursor)
	}
	u.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; F1-API-Ingestor/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("ingestor: start returned %d (continuing anyway)", resp.StatusCode)
	}
	return nil
}

// buildWSURL builds the WebSocket connection URL.
func buildWSURL(token, cursor string) string {
	u := &url.URL{
		Scheme: "wss",
		Host:   f1Host,
		Path:   "/signalr/connect",
	}
	q := url.Values{}
	q.Set("transport", "webSockets")
	q.Set("clientProtocol", clientProtocol)
	q.Set("connectionToken", token)
	q.Set("connectionData", connectionData)
	if cursor != "" {
		q.Set("messageId", cursor)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
