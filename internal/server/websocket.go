package server

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// WSHub manages WebSocket connections and broadcasts.
type WSHub struct {
	mu      sync.RWMutex
	clients map[*WSConn]bool
	msgCh   chan []byte
}

type WSConn struct {
	conn net.Conn
	mu   sync.Mutex
}

func NewWSHub() *WSHub {
	return &WSHub{
		clients: make(map[*WSConn]bool),
		msgCh:   make(chan []byte, 256),
	}
}

func (h *WSHub) Run() {
	for msg := range h.msgCh {
		h.mu.RLock()
		for c := range h.clients {
			c.WriteMessage(msg)
		}
		h.mu.RUnlock()
	}
}

func (h *WSHub) Broadcast(data interface{}) {
	msg, err := json.Marshal(data)
	if err != nil {
		return
	}
	select {
	case h.msgCh <- msg:
	default:
	}
}

func (h *WSHub) addClient(c *WSConn) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
}

func (h *WSHub) removeClient(c *WSConn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (c *WSConn) WriteMessage(data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	frame := make([]byte, 0, 2+8+len(data))
	frame = append(frame, 0x81) // text frame, FIN

	if len(data) < 126 {
		frame = append(frame, byte(len(data)))
	} else if len(data) < 65536 {
		frame = append(frame, 126, byte(len(data)>>8), byte(len(data)))
	} else {
		frame = append(frame, 127)
		for i := 7; i >= 0; i-- {
			frame = append(frame, byte(len(data)>>(i*8)))
		}
	}
	frame = append(frame, data...)
	c.conn.Write(frame)
}

func (c *WSConn) ReadMessage() ([]byte, error) {
	header := make([]byte, 2)
	if _, err := c.conn.Read(header); err != nil {
		return nil, err
	}

	masked := header[1]&0x80 != 0
	payloadLen := int(header[1] & 0x7F)

	if payloadLen == 126 {
		ext := make([]byte, 2)
		c.conn.Read(ext)
		payloadLen = int(ext[0])<<8 | int(ext[1])
	} else if payloadLen == 127 {
		ext := make([]byte, 8)
		c.conn.Read(ext)
		payloadLen = 0
		for i := 0; i < 8; i++ {
			payloadLen = payloadLen<<8 | int(ext[i])
		}
	}

	var maskKey []byte
	if masked {
		maskKey = make([]byte, 4)
		c.conn.Read(maskKey)
	}

	payload := make([]byte, payloadLen)
	n := 0
	for n < payloadLen {
		nn, err := c.conn.Read(payload[n:])
		if err != nil {
			return nil, err
		}
		n += nn
	}

	if masked {
		for i := 0; i < payloadLen; i++ {
			payload[i] ^= maskKey[i%4]
		}
	}

	// Check for close frame
	if header[0]&0x0F == 0x08 {
		return nil, fmt.Errorf("close frame")
	}

	return payload, nil
}

// handleWS upgrades HTTP to WebSocket using raw hijack (no gorilla dependency).
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") != "websocket" {
		http.Error(w, "expected websocket", http.StatusBadRequest)
		return
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}

	acceptKey := computeAcceptKey(key)

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket not supported", http.StatusInternalServerError)
		return
	}

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"
	bufrw.WriteString(resp)
	bufrw.Flush()

	wsConn := &WSConn{conn: conn}
	s.Hub.addClient(wsConn)

	// Send welcome
	welcome, _ := json.Marshal(map[string]string{"type": "connected", "engagement": s.EngName})
	wsConn.WriteMessage(welcome)

	// Read loop (keep connection alive, handle pings)
	go func() {
		defer func() {
			s.Hub.removeClient(wsConn)
			conn.Close()
		}()
		reader := bufio.NewReader(conn)
		_ = reader
		for {
			_, err := wsConn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()
}

func computeAcceptKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
