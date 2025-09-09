package ws

import (
	"encoding/json"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/configs"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/handlers"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/models"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"time"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Executes a handler and sends either success or error response back to client
func dispatch(conn *websocket.Conn, reqId int64, fn func(map[string]interface{}) (models.HandlerOK, models.HandlerError), req map[string]interface{}) {
	res, err := fn(req)
	if err.Code > 0 {
		handlers.SendWSError(conn, reqId, err.Type, err.Code, err.Data)
		return
	}
	handlers.SendWSResponse(conn, reqId, res.Type, res.Data)
	EmitServer(req, res.Type, res.Data)
}

// All WS routes mapped to handlers
var wsRoutes = map[string]func(*websocket.Conn, map[string]interface{}, int64){
	// Ping
	"ping": func(c *websocket.Conn, d map[string]interface{}, reqId int64) {
		dispatch(c, reqId, handlers.Ping, d)
	},
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	RegisterConn(conn)
	defer func() {
		UnregisterConn(conn)
		_ = conn.Close()
	}()

	// App token check
	if os.Getenv("DEBUG") != "1" {
		_, token, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket Read Error:", err)
			return
		}
		if string(token) != os.Getenv("APP_TOKEN") {
			handlers.SendWSError(conn, 0, "INVALID_APP_TOKEN", 1001, "")
			return
		}

	}

	// Handshake
	handlers.SendWSResponse(conn, 1, "handshake", map[string]interface{}{
		"apiVersion": configs.Version,
		"serverTime": time.Now().UTC().Format(time.RFC3339),
	})
	EmitToAnyEvent("heartbeat", handlers.ClientBattleIndex(handlers.BattleIndex))

	// Fill BattleIndex From DB
	if len(handlers.BattleIndex) == 0 {
		handlers.FillBattleIndex()
	}

	// Main loop
	var msg models.Request
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			handlers.SendWSError(conn, 0, "INVALID_JSON_BODY", 1002, "")
			continue
		}
		reqData, ok := msg.Data.(map[string]interface{})
		if !ok {
			handlers.SendWSError(conn, 0, "INVALID_DATA_FIELD_TYPE", 1003, "")
			continue
		}
		if configs.Debug {
			log.Println("Web Req:", msg.Type)
		}

		// Special case: bind
		if msg.Type == "bind" {
			handlers.SendWSResponse(conn, 1, "bind.ok", map[string]any{
				"at": time.Now().UTC().Format(time.RFC3339),
			})
			continue
		}

		// Dispatch via map
		if fn, found := wsRoutes[msg.Type]; found {
			fn(conn, reqData, msg.ReqID)
			continue
		}

		// Unknown route
		handlers.SendWSError(conn, 0, "UNKNOWN_ROUTE", 1010, map[string]any{"type": msg.Type})
	}
}
