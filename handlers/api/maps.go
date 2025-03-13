package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"

	"gitlab.com/glatteis/earthwalker/domain"
)

type Maps struct {
	MapStore             domain.MapStore
	ChallengeStore       domain.ChallengeStore
	ChallengeResultStore domain.ChallengeResultStore

	MapDeleteHandler MapDelete
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
		handler.MapDeleteHandler.ServeHTTP(w, r)
	default:
		sendError(w, "api/maps endpoint does not exist.", http.StatusNotFound)
	}
}

type MapDelete struct {
	Config domain.Config

	MapStore             domain.MapStore
	ChallengeStore       domain.ChallengeStore
	ChallengeResultStore domain.ChallengeResultStore
}

const mapDeleteNet = "127.0.0.0/8"

// TODO: generic local-only auth handler
func (handler MapDelete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if remote map deletion is allowed from the config
	allowRemote, err := strconv.ParseBool(handler.Config.AllowRemoteMapDeletion)
	if err != nil {
		sendError(w, "unable to parse AllowRemoteMapDeletion config value.", http.StatusInternalServerError)
		return
	}

	// If remote map deletion is not allowed, proceed with the IP validation
	if !allowRemote {
		// If the server is behind a proxy, check the X-Forwarded-For header for the real IP
		var clientIP string
		if handler.Config.IsBehindProxy == "True" {
			clientIP = r.Header.Get("X-Forwarded-For")
			if clientIP == "" {
				clientIP = r.RemoteAddr // Fall back to RemoteAddr if header is not present
			}
		} else {
			// If not behind a proxy, use the remote address directly
			clientIP, _, err = net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				sendError(w, "unable to split host from port in client IP address.", http.StatusInternalServerError)
				return
			}
		}

		// Check if the client's IP is in the AllowedIPs list
		allowed := false
		for _, allowedIP := range handler.Config.AllowedIPs {
			if clientIP == allowedIP {
				allowed = true
				break
			}
		}

		// If the IP is not allowed, return an Unauthorized error
		if !allowed {
			sendError(w, "you are not authorized to delete maps from this server.", http.StatusUnauthorized)
			return
		}
	}

	// Extract the mapID from the URL path
	mapID, _ := shiftPath(r.URL.Path)
	if len(mapID) == 0 || mapID == "/" {
		sendError(w, "missing map id", http.StatusBadRequest)
		return
	}

	// Proceed with deleting the map if everything is valid
	err = handler.deleteMap(mapID)
	if err != nil {
		sendError(w, "failed to delete map from store", http.StatusInternalServerError)
		log.Printf("Failed to delete map from store: %v\n", err)
		return
	}

	// Send a successful response after map deletion
	respJSON := "{\"data\": {\"message\": \"map with id: " + mapID + " deleted\"}}"
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(respJSON))
	if err != nil {
		log.Printf("Error writing response: %v\n", err)
	}
}

// TODO: consider a way of moving deletion chaining to the badgerdb package
func (handler MapDelete) deleteMap(mapID string) error {
	challengeIDs, err := handler.ChallengeStore.GetList(mapID)
	if err != nil {
		return fmt.Errorf("failed to get list of Challenge IDs: %v", err)
	}
	for _, challengeID := range challengeIDs {
		err = handler.ChallengeResultStore.DeleteAll(challengeID)
		if err != nil {
			return fmt.Errorf("failed to delete ChallengeResult: %v", err)
		}
	}
	err = handler.ChallengeStore.DeleteAll(mapID)
	err = handler.MapStore.Delete(mapID)
	if err != nil {
		return fmt.Errorf("failed to delete Map: %v", err)
	}
	return nil
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
