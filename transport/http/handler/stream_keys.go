package handler

import (
	"errors"
	"net/http"

	"lively/core"
	"lively/core/model"
	"lively/core/service"
	"lively/security/jwt"
	"lively/transport/http/codec"
)

type getSkByUserIDRes struct {
	StreamKey string `json:"stream_key"`
}

type StreamKeys struct {
	skService service.StreamKey
}

func NewStreamKeys(skService service.StreamKey) *StreamKeys {
	return &StreamKeys{skService}
}

func (h *StreamKeys) GetOfCurrentUser(w http.ResponseWriter, req *http.Request) {
	user, ok := jwt.GetUser(w, req)
	if !ok {
		return
	}
	var key model.StreamKey
	if err := h.skService.GetByUserID(req.Context(), user.ID, &key); err != nil {
		if errors.Is(err, core.ErrInactiveStreamKey) {
			http.Error(w, "", http.StatusNotFound)
		} else {
			sendHttpError(w, req, "get stream key", err)
		}
		return
	}
	codec.EncodeBody(w, http.StatusOK, &getSkByUserIDRes{StreamKey: key.Key})
}
