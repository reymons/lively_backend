package codec

import (
	"encoding/json"
	"log"
	"net/http"
)

// Decodes an http body and validates it.
//
// If the decoding process fails OR the body is invalid,
// automatically sends a 400-status response
//
// You must check the boolean flag that's returned to learn
// if the response has been sent (decoding failed)
//
// Example:
//
// body, ok := DecodeBody[MyBody](w, req)
//
//	if !ok {
//	    return
//	}
//
// Do something with the body, etc.
func DecodeBody[T Validator](w http.ResponseWriter, req *http.Request) (T, bool) {
	var body T
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return body, false
	}
	if errors := body.Valid(); len(errors) > 0 {
		if data, err := json.Marshal(errors); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(data)
		} else {
			http.Error(w, "failed to serialize validation errors", http.StatusInternalServerError)
		}
		return body, false
	}
	return body, true
}

// JSON-Encodes a body and automatically writes its contents to the response
// A user must not call any actions on the response writer afterwards
func EncodeBody[T any](w http.ResponseWriter, status int, v T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("ERROR: encode response body: %v", err)
	}
}
