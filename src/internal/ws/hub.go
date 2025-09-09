package ws

import (
	"encoding/json"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/events"
	"github.com/gorilla/websocket"
	"sync"
	"time"
)

type ConnInfo struct {
	Conn     *websocket.Conn
	UserID   int64
	SendChan chan []byte
}

var (
	regMu  sync.RWMutex
	byConn = make(map[*websocket.Conn]*ConnInfo)
	byUser = make(map[int64]map[*websocket.Conn]*ConnInfo)
)

// === Connection Lifecycle ===

func RegisterConn(c *websocket.Conn) {
	regMu.Lock()
	defer regMu.Unlock()

	if _, ok := byConn[c]; !ok {
		ci := &ConnInfo{
			Conn:     c,
			UserID:   0,
			SendChan: make(chan []byte, 64),
		}
		byConn[c] = ci
		go ci.startWriter()
	}
}

func UnregisterConn(c *websocket.Conn) {
	regMu.Lock()
	defer regMu.Unlock()

	ci, ok := byConn[c]
	if !ok {
		return
	}

	// Remove from user bucket
	if ci.UserID != 0 {
		if set, ok := byUser[ci.UserID]; ok {
			delete(set, c)
			if len(set) == 0 {
				delete(byUser, ci.UserID)
			}
		}
	}

	// Close writer
	close(ci.SendChan)
	delete(byConn, c)
}

func BindUser(c *websocket.Conn, userID int64) {
	regMu.Lock()
	defer regMu.Unlock()

	ci, ok := byConn[c]
	if !ok {
		return
	}

	// Remove from previous bucket
	if ci.UserID != 0 {
		if set, ok := byUser[ci.UserID]; ok {
			delete(set, c)
			if len(set) == 0 {
				delete(byUser, ci.UserID)
			}
		}
	}

	// Add to new bucket
	ci.UserID = userID
	if userID != 0 {
		if byUser[userID] == nil {
			byUser[userID] = make(map[*websocket.Conn]*ConnInfo)
		}
		byUser[userID][c] = ci
	}
}

// === Writer Goroutine ===

func (ci *ConnInfo) startWriter() {
	for msg := range ci.SendChan {
		_ = ci.Conn.WriteMessage(websocket.TextMessage, msg)
	}
}

// === Emit Core ===

func emitToTargets(targets []*ConnInfo, payload any) {
	if len(targets) == 0 {
		return
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	for _, ci := range targets {
		select {
		case ci.SendChan <- b:
		default:
			// Optional: drop or log overflow
		}
	}
}

// === Emit APIs ===

func EmitToUser(userID int64, payload any) {
	regMu.RLock()
	set := byUser[userID]
	var targets []*ConnInfo
	for _, ci := range set {
		targets = append(targets, ci)
	}
	regMu.RUnlock()
	emitToTargets(targets, payload)
}

func EmitToAllUsers(payload any) {
	regMu.RLock()
	var targets []*ConnInfo
	for _, set := range byUser {
		for _, ci := range set {
			targets = append(targets, ci)
		}
	}
	regMu.RUnlock()
	emitToTargets(targets, payload)
}

func EmitToAny(payload any) {
	regMu.RLock()
	var targets []*ConnInfo
	for _, ci := range byConn {
		targets = append(targets, ci)
	}
	regMu.RUnlock()
	emitToTargets(targets, payload)
}

func EmitToGuests(payload any) {
	regMu.RLock()
	var targets []*ConnInfo
	for _, ci := range byConn {
		if ci.UserID == 0 {
			targets = append(targets, ci)
		}
	}
	regMu.RUnlock()
	emitToTargets(targets, payload)
}

// === Event Helpers ===

func EmitToUserEvent(userID int64, eventType string, data any) {
	EmitToUser(userID, map[string]any{
		"type": eventType,
		"data": data,
		"at":   time.Now().UnixMilli(),
	})
}

func EmitToAllUsersEvent(eventType string, data any) {
	EmitToAllUsers(map[string]any{
		"type": eventType,
		"data": data,
		"at":   time.Now().UnixMilli(),
	})
}

func EmitToAnyEvent(eventType string, data any) {
	EmitToAny(map[string]any{
		"type": eventType,
		"data": data,
		"at":   time.Now().UnixMilli(),
	})
}

func EmitToGuestsEvent(eventType string, data any) {
	EmitToGuests(map[string]any{
		"type": eventType,
		"data": data,
		"at":   time.Now().UnixMilli(),
	})
}

func EmitServer(req map[string]interface{}, resType string, resData interface{}) {
	switch resType {
	case "test":
		// No Emit
	default:
		// Example: EmitToAnyEvent("heartbeat", handlers.LiveGame)
	}
}

// === Event Loop ===

func EmitEventLoop() {
	go func() {
		for ev := range events.Bus {
			switch ev.Target {
			case "all":
				EmitToAnyEvent(ev.Type, ev.Data)
			case "user":
				EmitToUserEvent(ev.UserID, ev.Type, ev.Data)
			case "allUsers":
				EmitToAllUsersEvent(ev.Type, ev.Data)
			case "guests":
				EmitToGuestsEvent(ev.Type, ev.Data)
			}
		}
	}()
}
