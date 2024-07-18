package handlers

import (
	"Go_Day03/src/types"
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"strconv"
)

const (
	pageSize = 10
)

type HandlePlaces struct {
	Name     string
	Total    int
	Places   []types.Place
	Page     int
	LastPage int
	PrevPage int
	NextPage int
}

type Recommendation struct {
	Name   string        `json:"name"`
	Places []types.Place `json:"places"`
}

func handleGetPlaces(w http.ResponseWriter, r *http.Request, client types.DataStore, indexName string) (*HandlePlaces, error) {
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		pageStr = "1"
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		http.Error(w, "Invalid page number", http.StatusBadRequest)
		return nil, err
	}

	places, total, err := client.GetPlaces(pageSize, (page-1)*pageSize)
	if err != nil {
		http.Error(w, "Invalid 'page' value: "+pageStr, http.StatusBadRequest)
		return nil, err
	}

	lastPage := (total + pageSize - 1) / pageSize

	data := &HandlePlaces{
		Name:     indexName,
		Places:   places,
		Total:    total,
		Page:     page,
		LastPage: lastPage,
	}

	if page > 1 {
		data.PrevPage = page - 1
	}

	if page < lastPage {
		data.NextPage = page + 1
	}

	return data, nil
}

func HandleGetPlacesHTML(w http.ResponseWriter, r *http.Request, client types.DataStore, tmpl *template.Template) {
	data, err := handleGetPlaces(w, r, client, "Places")
	if err != nil {
		return
	}

	if err = tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

func HandleGetPlacesAPI(w http.ResponseWriter, r *http.Request, client types.DataStore) {
	data, err := handleGetPlaces(w, r, client, "Places")
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err = json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Error rendering JSON", http.StatusInternalServerError)
	}
}

func LoadTemplate(filename string) (*template.Template, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return template.New("places").Funcs(template.FuncMap{
		"sub": func(a, b int) int { return a - b },
		"add": func(a, b int) int { return a + b },
		"div": func(a, b int) int { return a / b },
	}).Parse(string(data))
}

func HandleRecommendAPI(w http.ResponseWriter, r *http.Request, client types.DataStore) {
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")
	if latStr == "" || lonStr == "" {
		http.Error(w, "Missing latitude or longitude", http.StatusBadRequest)
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		http.Error(w, "Invalid latitude", http.StatusBadRequest)
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		http.Error(w, "Invalid longitude", http.StatusBadRequest)
		return
	}

	places, err := client.GetNearbyPlaces(lat, lon)
	if err != nil {
		http.Error(w, "Error fetching recommendations", http.StatusInternalServerError)
		return
	}

	response := Recommendation{
		Name:   "Recommendation",
		Places: places,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
