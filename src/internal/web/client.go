package web

import (
	"encoding/json"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/configs"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/handlers"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/models"
	"log"
	"net/http"
	"os"
)

// dispatchWeb runs a handler and writes either error or success to the HTTP response.
func dispatchWeb(w http.ResponseWriter, fn func(map[string]interface{}) (models.HandlerOK, models.HandlerError), req map[string]interface{}) {
	res, err := fn(req)
	if err.Code > 0 {
		SendError(w, err.Type, err.Code, err.Data)
		return
	}
	SendResponse(w, res.Type, res.Data)
}

// All POST routes mapped to handlers
var postRoutes = map[string]func(map[string]interface{}) (models.HandlerOK, models.HandlerError){
	// Ping
	"ping": handlers.Ping,
}

func HandleHTTP(w http.ResponseWriter, r *http.Request) {
	// App token validation
	token := r.Header.Get("Authorization")
	if token != "Bearer "+os.Getenv("APP_TOKEN") {
		SendError(w, "INVALID_APP_TOKEN", 1001)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse envelope
	var req models.Payload
	var msg models.Request

	if err := json.NewDecoder(r.Body).Decode(&req.Payload); err != nil {
		SendError(w, "INVALID_JSON_BODY", 1002)
		return
	}
	if err := json.Unmarshal(req.Payload, &msg); err != nil {
		SendError(w, "INVALID_JSON_BODY", 1002)
		return
	}

	reqData, ok := msg.Data.(map[string]interface{})
	if !ok {
		SendError(w, "INVALID_DATA_FIELD_TYPE", 1003)
		return
	}

	if configs.Debug {
		log.Println("HTTP Req:", msg.Type)
	}

	switch r.Method {
	case http.MethodPost:
		if fn, ok := postRoutes[msg.Type]; ok {
			dispatchWeb(w, fn, reqData)
			return
		}
		SendError(w, "UNKNOWN_ROUTE", 1010)

	case http.MethodPut:
		SendError(w, "METHOD_NOT_ALLOWED", 1004)

	case http.MethodDelete:
		SendError(w, "METHOD_NOT_ALLOWED", 1004)

	default:
		SendError(w, "METHOD_NOT_ALLOWED", 1004)
	}
}
