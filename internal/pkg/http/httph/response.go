package httph

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"
)

func SendJSON(w http.ResponseWriter, status int, body any) {
	data, err := json.Marshal(body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err = w.Write(data); err != nil {
		log.Error().Err(err).Msg("Failed to write response")
	}
}
func SendEmpty(w http.ResponseWriter, status int) { w.WriteHeader(status) }
func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	status, message := http.StatusInternalServerError, "internal server error"
	var coded interface {
		error
		HTTPStatus() int
	}
	if errors.As(err, &coded) {
		status, message = coded.HTTPStatus(), coded.Error()
	}
	ErrorApply(r, err)
	ErrorApplyStatusCode(r, status)
	SendJSON(w, status, map[string]string{"error": message})
}
