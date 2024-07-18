package types

type Place struct {
	ID       string   `json:":id"`
	Name     string   `json:"name"`
	Address  string   `json:"address"`
	Phone    string   `json:"phone"`
	Location GeoPoint `json:"location"`
}

type GeoPoint struct {
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

type DataStore interface {
	GetPlaces(limit, offset int) ([]Place, int, error)
	GetNearbyPlaces(lat, lon float64) ([]Place, error)
}
