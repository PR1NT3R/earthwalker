// Package handlers is an enumeration of backend handlers
package handlers

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"gitlab.com/glatteis/earthwalker/domain"
)

const challengeCookieName = "earthwalker_lastChallenge"
const resultCookiePrefix = "earthwalker_lastResult_"

// A Play is a context to ServeHTTP on
type Play struct {
	ChallengeStore       domain.ChallengeStore
	ChallengeResultStore domain.ChallengeResultStore
	Config               domain.Config
}

func (handler Play) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	challengeID, err := getChallengeID(r)
	if err != nil {
		http.Error(w, "no challengeID in request URL or cookies", http.StatusBadRequest)
		log.Printf("No challengeID in request URL or cookies: %v", err)
	}
	resultID, err := getResultID(r, challengeID)
	if err != nil {
		// no result ID, redirect to /join?id=<challengeID>
		http.Redirect(w, r, "/join?id="+challengeID, http.StatusTemporaryRedirect)
		return
	}
	result, err := handler.ChallengeResultStore.Get(resultID)
	if err != nil {
		http.Error(w, "failed to retrieve result", http.StatusInternalServerError)
		log.Printf("Failed to retrieve result with ID '%s' from store: %v", resultID, err)
	}
	challenge, err := handler.ChallengeStore.Get(result.ChallengeID)
	if err != nil {
		http.Error(w, "failed to retrieve challenge", http.StatusInternalServerError)
		log.Printf("Failed to retrieve challenge with ID '%s' from store: %v", result.ChallengeID, err)
	}
	// user has already finished this challenge, redirect to /summary
	if len(result.Guesses) >= len(challenge.Places) {
		http.Redirect(w, r, "/summary", http.StatusTemporaryRedirect)
		return
	}
	// (re)set cookies
	http.SetCookie(w, &http.Cookie{
		Name:     challengeCookieName,
		Value:    result.ChallengeID,
		MaxAge:   172800,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     resultCookiePrefix + result.ChallengeID,
		Value:    result.ChallengeResultID,
		MaxAge:   172800,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
	log.Println(challenge.Places[len(result.Guesses)].Location)
	// TODO: FIXME: this fails catastrophically if the player has already
	//              completed the challenge and tries to navigate back to /play
	handler.ServeLocation(challenge.Places[len(result.Guesses)].Location, w, r)
}

func getChallengeID(r *http.Request) (string, error) {
	// try url params first
	ids, ok := r.URL.Query()["id"]
	if ok && len(ids[0]) > 0 {
		return ids[0], nil
	}
	// if no id param, look in cookies
	challengeCookie, err := r.Cookie(challengeCookieName)
	if err != nil {
		return "", fmt.Errorf("no challenge cookie found: %v", err)
	}
	return challengeCookie.Value, nil
}

func getResultID(r *http.Request, challengeID string) (string, error) {
	resultCookie, err := r.Cookie(resultCookiePrefix + challengeID)
	if err != nil {
		return "", fmt.Errorf("no result cookie found: %v", err)
	}
	return resultCookie.Value, nil
}

func (handler Play) modifyMainPage(target string, w http.ResponseWriter, r *http.Request) {
	res, err := http.Get(target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	bodyAsString := string(body)

	insertBody, err := ioutil.ReadFile(handler.Config.StaticPath + "/public/modify_frontend/modify.html")
	if err != nil {
		log.Fatal(err)
	}

	replacedBody := strings.Replace(bodyAsString, "<head>", "<head> "+string(insertBody), 1)
	w.Write([]byte(filterUrls(replacedBody)))
}

func modifyInformation(target string, w http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequest("GET", target, nil)
	req.Header.Add("User-Agent", r.Header.Get("User-Agent"))
	req.Header.Add("Accept", r.Header.Get("Accept"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	if strings.Contains(target, "photometa") {
		body = []byte(filterPhotometa(string(body)))
	} else {
		body = []byte(filterUrls(string(body)))
	}

	for header := range res.Header {
		w.Header().Add(header, res.Header.Get(header))
	}
	w.Write(body)
}

func floatToString(number float64) string {
	return strconv.FormatFloat(number, 'f', 14, 64)
}

// buildURL builds google street view urls from coordinates
func buildURL(location domain.Coords) string {

	template := "https://www.google.com/maps/@%f,%f,3a,90y,0h,90t/data=!3m7!1e1!3m5!1s%s!2e0!3e11!7i3512!8i894?hl=en"
	return fmt.Sprintf(template, location.Lat, location.Lng, location.PanoID)

	// 	baseURL, err := url.Parse("https://www.google.com/maps")
	// 	if err != nil {
	// 		log.Fatal("Failed while parsing static gmaps url", err)
	// 	}
	// 	query := baseURL.Query()
	// 	// see https://stackoverflow.com/questions/387942/google-street-view-url
	// 	// for a reverse-engineering of the parameters

	// 	// the layer must be set to c (the street view layer)
	// 	query.Set("layer", "c")
	// 	// latitude and longitude go into parameter cbll
	// 	query.Set("cbll", floatToString(location.Lat)+","+floatToString(location.Lng))

	// 	baseURL.RawQuery = query.Encode()
	// 	fmt.Println(baseURL.String())
	// 	return baseURL.String()
}

// ServeLocation serves a specific location to the user.
func (handler Play) ServeLocation(l domain.Coords, w http.ResponseWriter, r *http.Request) {
	mapsURL := buildURL(l)
	handler.modifyMainPage(mapsURL, w, r)
}

// ServeGoogle is a proxy to google
func ServeGoogle(w http.ResponseWriter, r *http.Request) {
	fullURL := r.URL
	fullURL.Host = "www.google.com"
	fullURL.Scheme = "https"

	modifyInformation(fullURL.String(), w, r)
}
