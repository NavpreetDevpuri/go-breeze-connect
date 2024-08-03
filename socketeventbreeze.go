package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type SocketEventBreeze struct {
	namespace      string
	breeze         *BreezeInstance
	conn           *websocket.Conn
	tokenlist      map[string]bool
	ohlcstate      map[string]bool
	authentication bool
	mu             sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewSocketEventBreeze(namespace string, breeze *BreezeInstance) *SocketEventBreeze {
	ctx, cancel := context.WithCancel(context.Background())
	return &SocketEventBreeze{
		namespace:      namespace,
		breeze:         breeze,
		tokenlist:      make(map[string]bool),
		ohlcstate:      make(map[string]bool),
		authentication: true,
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (seb *SocketEventBreeze) Connect(hostname string, isOHLCStream bool, strategyFlag bool) error {
	var err error
	seb.conn, _, err = websocket.Dial(seb.ctx, hostname, nil)
	if err != nil {
		return err
	}

	auth := map[string]string{
		"user":  seb.breeze.UserID,
		"token": seb.breeze.SessionKey,
	}

	if err := wsjson.Write(seb.ctx, seb.conn, auth); err != nil {
		return err
	}

	go seb.readMessages()
	return nil
}

func (seb *SocketEventBreeze) readMessages() {
	for {
		var msg map[string]interface{}
		if err := wsjson.Read(seb.ctx, seb.conn, &msg); err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return
			}
			log.Println("readMessages error:", err)
			continue
		}

		seb.handleMessage(msg)
	}
}

func (seb *SocketEventBreeze) handleMessage(msg map[string]interface{}) {
	if eventType, ok := msg["event"]; ok {
		switch eventType {
		case "order":
			data, err := json.Marshal(msg["data"])
			if err == nil {
				seb.onMessage(data)
			}
		case "ohlc":
			data, err := json.Marshal(msg["data"])
			if err == nil {
				seb.onOHLCStream(data)
			}
		}
	}
}

func (seb *SocketEventBreeze) OnDisconnect() {
	seb.cancel()
	seb.conn.Close(websocket.StatusNormalClosure, "transport close")
}

func (seb *SocketEventBreeze) Notify() {
	seb.conn.Write(seb.ctx, websocket.MessageText, []byte(`{"type": "notify"}`))
}

func (seb *SocketEventBreeze) onMessage(data []byte) {
	parsedData := seb.breeze.ParseData(data)
	if symbol, ok := parsedData["symbol"]; ok && symbol != nil {
		stockData := seb.breeze.GetDataFromStockTokenValue(symbol.(string))
		for k, v := range stockData {
			parsedData[k] = v
		}
	}
	if seb.breeze.OnTicks != nil {
		seb.breeze.OnTicks(parsedData)
	}
	if seb.breeze.OnTicks2 != nil {
		seb.breeze.OnTicks2(parsedData)
	}
}

func (seb *SocketEventBreeze) onOHLCStream(data []byte) {
	parsedData := seb.breeze.ParseOHLCData(data)
	seb.breeze.OnTicks(parsedData)
}

func (seb *SocketEventBreeze) RewatchOHLC() {
	seb.mu.Lock()
	defer seb.mu.Unlock()
	for room := range seb.ohlcstate {
		seb.conn.Write(seb.ctx, websocket.MessageText, []byte(`{"type": "join", "room": "`+room+`"}`))
	}
}

func (seb *SocketEventBreeze) WatchStreamData(data, channel string) error {
	if seb.conn != nil {
		if !seb.ohlcstate[data] {
			seb.ohlcstate[data] = true
		}
		seb.conn.Write(seb.ctx, websocket.MessageText, []byte(`{"type": "join", "data": "`+data+`"}`))
		return nil
	}
	return fmt.Errorf("OHLC_SOCKET_CONNECTION_DISCONNECTED")
}

func (seb *SocketEventBreeze) Rewatch() {
	seb.Notify()
	seb.mu.Lock()
	defer seb.mu.Unlock()
	if len(seb.tokenlist) > 0 {
		tokens := make([]string, 0, len(seb.tokenlist))
		for token := range seb.tokenlist {
			tokens = append(tokens, token)
		}
		seb.conn.Write(seb.ctx, websocket.MessageText, []byte(`{"type": "join", "tokens": "`+fmt.Sprintf("%v", tokens)+`"}`))
	}
}

func (seb *SocketEventBreeze) Watch(data interface{}) error {
	if seb.conn != nil {
		switch v := data.(type) {
		case []string:
			for _, entry := range v {
				seb.tokenlist[entry] = true
			}
		case string:
			seb.tokenlist[v] = true
		}
		seb.conn.Write(seb.ctx, websocket.MessageText, []byte(`{"type": "join", "data": "`+fmt.Sprintf("%v", data)+`"}`))
		return nil
	}
	return fmt.Errorf("LIVESTREAM_SOCKET_CONNECTION_DISCONNECTED")
}

func (seb *SocketEventBreeze) Unwatch(data interface{}) {
	switch v := data.(type) {
	case []string:
		for _, entry := range v {
			delete(seb.tokenlist, entry)
		}
	case string:
		delete(seb.tokenlist, v)
	}
	seb.mu.Lock()
	defer seb.mu.Unlock()
	toBeRemoved := []string{}
	for room := range seb.ohlcstate {
		if room == data {
			toBeRemoved = append(toBeRemoved, room)
		}
	}
	for _, room := range toBeRemoved {
		delete(seb.ohlcstate, room)
	}
	seb.conn.Write(seb.ctx, websocket.MessageText, []byte(`{"type": "leave", "data": "`+fmt.Sprintf("%v", data)+`"}`))
}
