package angelapi

import (
	"net/http"
	"strconv"
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

	return response["data"], nil
}
