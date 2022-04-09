package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"gitlab.com/glatteis/earthwalker/domain"
)

type Maps struct {
	MapStore domain.MapStore
}

func (handler Maps) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mapID, _ := shiftPath(r.URL.Path)
		if len(mapID) == 0 || mapID == "/" {
			sendError(w, "missing map id", http.StatusBadRequest)
			return
		}
		// return MapStore.GetAll if path is /all
		if mapID == "all" {
			foundMaps, err := handler.MapStore.GetAll()
			if err != nil {
				sendError(w, "failed to get maps from store", http.StatusInternalServerError)
				log.Printf("Failed to get maps from store: %v\n", err)
				return
			}
			json.NewEncoder(w).Encode(foundMaps)
			return
		}
		foundMap, err := handler.MapStore.Get(mapID)
		if err != nil {
			sendError(w, "failed to get map from store", http.StatusInternalServerError)
			log.Printf("Failed to get map from store: %v\n", err)
			return
		}
		json.NewEncoder(w).Encode(foundMap)
	case http.MethodPost:
		newMap, err := mapFromRequest(r)
		if err != nil {
			sendError(w, "failed to create map from request", http.StatusInternalServerError)
			log.Printf("Failed to create map from request: %v\n", err)
			return
		}
		err = handler.MapStore.Insert(newMap)
		if err != nil {
			sendError(w, "failed to insert map into store", http.StatusInternalServerError)
			log.Printf("Failed to insert map into store: %v\n", err)
			return
		}
		json.NewEncoder(w).Encode(newMap)
	case http.MethodDelete:
		mapID, _ := shiftPath(r.URL.Path)
		if len(mapID) == 0 || mapID == "/" {
			sendError(w, "missing map id", http.StatusBadRequest)
			return
		}
		err := handler.MapStore.Delete(mapID)
		if err != nil {
			sendError(w, "failed to delete map from store", http.StatusInternalServerError)
			log.Printf("Failed to delete map from store: %v\n", err)
			return
		}
		// TODO: FIXME: extremely awk success response
		respJSON := "{\"data\": {\"message\": \"map with id: " + mapID + " deleted\"}}"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(respJSON))
		if err != nil {
			log.Printf("Error writing response: %v\n", err)
		}
	default:
		sendError(w, "api/maps endpoint does not exist.", http.StatusNotFound)
	}
}

func mapFromRequest(r *http.Request) (domain.Map, error) {
	newMap := domain.Map{}
	err := json.NewDecoder(r.Body).Decode(&newMap)
	if err != nil {
		return newMap, fmt.Errorf("failed to decode newMap from request: %v", err)
	}
	// we want to make sure we don't take the ID from the client request
	newMap.MapID = domain.RandAlpha(10)
	return newMap, nil
}
