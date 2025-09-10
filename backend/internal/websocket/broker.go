package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Yusufzhafir/go-orderbook/backend/pkg/model"
	"github.com/gorilla/websocket"
)

const (
	writeWait           = 10 * time.Second
	pongWait            = 60 * time.Second
	pingPeriod          = (pongWait * 9) / 10
	maxMessageSize      = 512 * 1024 // 512 KB
	defaultSendBuf      = 256
	defaultPublishBuf   = 4096
	maxConsecutiveDrops = 50
)

// Trade is the message payload for a running trade.
type Trade struct {
	Symbol string         `json:"symbol"`
	Price  model.Price    `json:"price"`
	Qty    model.Quantity `json:"qty"`
	Side   string         `json:"side"` // "buy" / "sell"
	Ts     int64          `json:"ts"`   // unix ms
	Seq    uint64         `json:"seq,omitempty"`
}

type publishMsg struct {
	Topic string
	Data  []byte
}

type subscription struct {
	client *Client
	topic  string
}

// Hub manages clients, subscriptions and publishes.
type Hub struct {
	register    chan *Client
	unregister  chan *Client
	subscribe   chan subscription
	unsubscribe chan subscription
	publish     chan publishMsg

	clients map[*Client]struct{}
	topics  map[string]map[*Client]struct{}

	// Configuration
	sendBuf int

	// simple metrics
	publishDrops uint64

	logger *log.Logger
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	subscribed map[string]struct{}

	// consecutive drops counter: if it grows too large we evict the client
	drops int
}

// NewHub creates a Hub with reasonable defaults. Provide a logger or nil.
func NewHub(logger *log.Logger) *Hub {
	if logger == nil {
		logger = log.Default()
	}
	return &Hub{
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		subscribe:   make(chan subscription),
		unsubscribe: make(chan subscription),
		publish:     make(chan publishMsg, defaultPublishBuf),
		clients:     make(map[*Client]struct{}),
		topics:      make(map[string]map[*Client]struct{}),
		sendBuf:     defaultSendBuf,
		logger:      logger,
	}
}

// Run runs the hub event loop. Call as: go hub.Run(ctx).
// The hub stops when ctx is cancelled.
func (h *Hub) Run(ctx context.Context) {
	h.logger.Println("ws hub started")
	for {
		select {
		case c := <-h.register:
			h.clients[c] = struct{}{}

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				for t := range c.subscribed {
					if subs := h.topics[t]; subs != nil {
						delete(subs, c)
						if len(subs) == 0 {
							delete(h.topics, t)
						}
					}
				}
				close(c.send)
			}

		case sub := <-h.subscribe:
			subs := h.topics[sub.topic]
			if subs == nil {
				subs = make(map[*Client]struct{})
				h.topics[sub.topic] = subs
			}
			subs[sub.client] = struct{}{}
			sub.client.subscribed[sub.topic] = struct{}{}

		case sub := <-h.unsubscribe:
			if subs := h.topics[sub.topic]; subs != nil {
				delete(subs, sub.client)
				if len(subs) == 0 {
					delete(h.topics, sub.topic)
				}
			}
			delete(sub.client.subscribed, sub.topic)

		case p := <-h.publish:
			if p.Topic == "" {
				// broadcast to all clients
				for c := range h.clients {
					select {
					case c.send <- p.Data:
					default:
						atomic.AddUint64(&h.publishDrops, 1)
						c.drops++
						if c.drops > maxConsecutiveDrops {
							h.logger.Printf(
								"evicting slow client after %d drops", c.drops,
							)
							delete(h.clients, c)
							for t := range c.subscribed {
								if s := h.topics[t]; s != nil {
									delete(s, c)
									if len(s) == 0 {
										delete(h.topics, t)
									}
								}
							}
							close(c.send)
							_ = c.conn.Close()
						}
					}
				}
			} else {
				// publish to a topic (symbol)
				if subs := h.topics[p.Topic]; subs != nil {
					for c := range subs {
						select {
						case c.send <- p.Data:
						default:
							atomic.AddUint64(&h.publishDrops, 1)
							c.drops++
							if c.drops > maxConsecutiveDrops {
								h.logger.Printf(
									"evicting slow client after %d drops", c.drops,
								)
								delete(h.clients, c)
								for t := range c.subscribed {
									if s := h.topics[t]; s != nil {
										delete(s, c)
										if len(s) == 0 {
											delete(h.topics, t)
										}
									}
								}
								close(c.send)
								_ = c.conn.Close()
							}
						}
					}
				}
			}

		case <-ctx.Done():
			h.logger.Println("ws hub shutting down")
			// clean up clients
			for c := range h.clients {
				close(c.send)
				_ = c.conn.Close()
				delete(h.clients, c)
			}
			return
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// In prod, check origin and require auth.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWS upgrades the request and registers a client.
// You can pass initial symbols via ?symbols=BTC-USD,ETH-USD
func ServeWS(h *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "upgrade failed", http.StatusBadRequest)
		return
	}

	client := &Client{
		hub:        h,
		conn:       conn,
		send:       make(chan []byte, h.sendBuf),
		subscribed: make(map[string]struct{}),
	}

	// optional: subscribe from query param
	if s := r.URL.Query().Get("symbols"); s != "" {
		for _, sym := range strings.Split(s, ",") {
			sym = strings.TrimSpace(sym)
			if sym == "" {
				continue
			}
			client.subscribed[sym] = struct{}{}
		}
	}

	// register then register subscriptions
	h.register <- client
	for sym := range client.subscribed {
		h.subscribe <- subscription{client: client, topic: sym}
	}

	// start reader and writer
	go client.writePump()
	go client.readPump()
}

// readPump reads control/command messages from the client
// and turns them into subscribe/unsubscribe requests.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseAbnormalClosure,
			) {
				c.hub.logger.Printf("read error: %v", err)
			}
			return
		}

		// any incoming activity -> reset drops counter
		c.drops = 0

		var cmd struct {
			Type   string `json:"type"`   // "subscribe" | "unsubscribe"
			Symbol string `json:"symbol"` // e.g. "BTC-USD"
		}
		if err := json.Unmarshal(message, &cmd); err != nil {
			c.hub.logger.Printf("invalid client msg: %v", err)
			continue
		}

		switch cmd.Type {
		case "subscribe":
			if cmd.Symbol != "" {
				c.hub.subscribe <- subscription{client: c, topic: cmd.Symbol}
			}
		case "unsubscribe":
			if cmd.Symbol != "" {
				c.hub.unsubscribe <- subscription{client: c, topic: cmd.Symbol}
			}
		default:
			// unknown: ignore or extend protocol
		}
	}
}

// writePump serializes all writes to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// hub closed the channel
				_ = c.conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				)
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				_ = w.Close()
				return
			}

			// batch queued messages into same frame
			n := len(c.send)
			for i := 0; i < n; i++ {
				if msg := <-c.send; msg != nil {
					if _, err := w.Write([]byte("\n")); err != nil {
						break
					}
					if _, err := w.Write(msg); err != nil {
						break
					}
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// PublishTrade publishes a trade to subscribers of t.Symbol.
// Non-blocking: if the hub publish buffer is full, the trade is dropped.
func (h *Hub) PublishTrade(t Trade) {
	t.Seq = nextSeq(t.Symbol)
	payload := struct {
		Type  string `json:"type"`
		Trade Trade  `json:"trade"`
	}{"trade", t}
	b, err := json.Marshal(payload)
	if err != nil {
		h.logger.Printf("marshal trade: %v", err)
		return
	}

	select {
	case h.publish <- publishMsg{Topic: t.Symbol, Data: b}:
	default:
		// avoid blocking producers; track drops
		atomic.AddUint64(&h.publishDrops, 1)
		h.logger.Println("publish channel full, dropping trade")
	}
}

// Stats returns simple metrics (clients count and publish drops).
func (h *Hub) Stats() (clients int, drops uint64) {
	clients = len(h.clients)
	drops = atomic.LoadUint64(&h.publishDrops)
	return
}
