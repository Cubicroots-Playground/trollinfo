package shiftnotifier

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"text/template"
	"time"

	_ "embed"
)

//go:embed template/landscape.html
var landscapeTemplate string

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

func (service *service) serveHumanPortrait(w http.ResponseWriter, r *http.Request) {
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

func (service *service) serveHumanLandscape(w http.ResponseWriter, r *http.Request) {
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

	defaultTZ, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		defaultTZ = time.Local
	}

	timeStr := service.latestDiffs.ReferenceTime.
		Add(service.config.NotifyBeforeShiftStart).
		In(defaultTZ).
		Format("Mon, 15:04")

	tmpl, err := template.New("landscape").Parse(landscapeTemplate)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	err = tmpl.Execute(w, map[string]any{
		"data":            service.latestDiffs,
		"refresh_seconds": r.URL.Query().Get("refresh_seconds"),
		"shift_time":      timeStr,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
}
