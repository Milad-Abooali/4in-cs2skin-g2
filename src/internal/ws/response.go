package ws

import (
	"encoding/json"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/configs"
	"github.com/Milad-Abooali/4in-cs2skin-g2/src/internal/models"
	"log"
)

// SendResponse WebSocket response (success)
func SendResponse(ci *ConnInfo, reqId int64, resType string, data interface{}) {
	resp := models.ReqRes{
		ReqID:  reqId,
		Type:   resType,
		Status: 1,
		Data:   data,
	}
	b, _ := json.Marshal(resp)
	select {
	case ci.SendChan <- b:
	default:
		if configs.Debug {
			log.Printf("SendResponse overflow for user %d", ci.UserID)
		}
	}
}

// SendError WebSocket response (error)
func SendError(ci *ConnInfo, reqId int64, resType string, eCode int, eExtra ...any) {
	if configs.Debug {
		log.Printf("Error %d | %s", eCode, resType)
	}
	resp := models.ReqRes{
		ReqID:  reqId,
		Type:   resType,
		Status: 0,
		Error:  eCode,
		Data:   eExtra,
	}
	b, _ := json.Marshal(resp)
	select {
	case ci.SendChan <- b:
	default:
		if configs.Debug {
			log.Printf("SendError overflow for user %d", ci.UserID)
		}
	}
}
