package shiftnotifier

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

func (service *service) requireToken(r *http.Request) error {
	t := r.URL.Query().Get("token")
	if strings.TrimSpace(t) != service.config.Token {
		return errors.New("invalid auth")
	}
	return nil
}

func (service *service) serveJSONData(w http.ResponseWriter, r *http.Request) {
	err := service.requireToken(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
		return
	}

	data, err := json.Marshal(service.latestDiffs)
	if err != nil {
		slog.Error("failed marshaling data", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
		return
	}
	_, _ = w.Write(data)
}

func (service *service) serveHumanData(w http.ResponseWriter, r *http.Request) {
	err := service.requireToken(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
		return
	}

	if service.latestDiffs == nil {
		_, _ = w.Write([]byte("no data"))
		return
	}

	_, html := service.diffToMessage(service.latestDiffs)

	if refreshSeconds := r.URL.Query().Get("refresh_seconds"); refreshSeconds != "" {
		html = `<meta http-equiv="refresh" content="` + refreshSeconds + `">` + html
	}

	html = `<html>
<style>
body {
	background:#111;
	color:darkgrey;
	font-family: sans-serif;
}
</style>
	` + html + "</html>"

	_, _ = w.Write([]byte(html))
}
