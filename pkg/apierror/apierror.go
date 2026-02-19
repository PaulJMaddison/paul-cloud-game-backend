package apierror

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Write(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{Code: code, Message: message})
}
