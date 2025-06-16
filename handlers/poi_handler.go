package handlers

import (
	"encoding/json"
	"go-server/middleware"
	"go-server/models"
	"go-server/services"
	"go-server/utils/errors"
	"net/http"
	"strconv"
)

type POIHandler struct {
	geoService *services.GeoService
}

type NearbyPOIResponse struct {
	NearbyPOIs []models.POI `json:"nearby_pois"`
	Count      int          `json:"count"`
	Lat        float64      `json:"lat"`
	Lon        float64      `json:"lon"`
	Radius     float64      `json:"radius"`
}

func NewPOIHandler(geoService *services.GeoService) *POIHandler {
	return &POIHandler{geoService: geoService}
}

func (h *POIHandler) GetNearbyPOIs(w http.ResponseWriter, r *http.Request) {
	lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	if err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}
	lon, err := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}
	radius, err := strconv.ParseFloat(r.URL.Query().Get("radius"), 64)
	if err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}
	poiType := r.URL.Query().Get("type")

	pois, err := h.geoService.FindNearbyPOIs(r.Context(), lat, lon, radius, poiType)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Create response object
	response := NearbyPOIResponse{
		NearbyPOIs: pois,
		Count:      len(pois),
		Lat:        lat,
		Lon:        lon,
		Radius:     radius,
	}
	json.NewEncoder(w).Encode(response)
}
