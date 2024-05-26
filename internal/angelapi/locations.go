package angelapi

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

func (service *service) ListLocations(_ *ListLocationsOpts) ([]Location, error) {
	response := map[string][]Location{}
	err := service.makeRequest(http.MethodGet, "/api/v0-beta/locations", nil, &response)
	if err != nil {
		return nil, err
	}

	return response["data"], nil
}

func (service *service) ListShiftsInLocation(locationID int64, _ *ListShiftsInLocationOpts) ([]Shift, error) {
	response := map[string][]Shift{}
	err := service.makeRequest(http.MethodGet, "/api/v0-beta/locations/"+strconv.Itoa(int(locationID))+"/shifts", nil, &response)
	if err != nil {
		return nil, err
	}

	// Parse time.
	for i := range response["data"] {
		startsAt, err := time.Parse(timeFormatISO8601, response["data"][i].StartsAtRaw)
		if err != nil {
			slog.Error("failed to parse time", "time", response["data"][i].StartsAtRaw, "error", err.Error())
		}
		endsAt, err := time.Parse(timeFormatISO8601, response["data"][i].EndsAtRaw)
		if err != nil {
			slog.Error("failed to parse time", "time", response["data"][i].EndsAtRaw, "error", err.Error())
		}

		response["data"][i].StartsAt = startsAt
		response["data"][i].EndsAt = endsAt
	}

	return response["data"], nil
}
