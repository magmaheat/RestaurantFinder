package db

import (
	"Go_Day03/src/types"
	"context"
	"encoding/csv"
	"encoding/json"
	"github.com/olivere/elastic/v7"
	"log"
	"os"
	"strconv"
	"sync"
)

type Place struct {
	ID       string           `json:":id"`
	Name     string           `json:"name"`
	Address  string           `json:"address"`
	Phone    string           `json:"phone"`
	Location elastic.GeoPoint `json:"location"`
}

type ElasticStore struct {
	Client *elastic.Client
	Index  string
}

func (es *ElasticStore) GetNearbyPlaces(lat, lon float64) ([]types.Place, error) {
	ctx := context.Background()

	searchResult, err := es.Client.Search().
		Index(es.Index).
		Query(elastic.NewMatchAllQuery()).
		SortBy(elastic.NewGeoDistanceSort("location").
			Point(lat, lon).
			Asc().
			Unit("km").
			DistanceType("arc").
			IgnoreUnmapped(true)).
		Size(3).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	var places []types.Place
	for _, hit := range searchResult.Hits.Hits {
		var place types.Place
		err := json.Unmarshal(hit.Source, &place)
		if err != nil {
			continue
		}
		places = append(places, place)
	}

	return places, nil
}

func NewElasticStore() (*ElasticStore, error) {
	client, err := elastic.NewClient(elastic.SetURL("http://localhost:9200"))
	if err != nil {
		log.Fatalf("Error creating the client: ")
		return nil, err
	}
	return &ElasticStore{Client: client}, nil
}

func (es *ElasticStore) LoadData(pathData string) {
	places, err := es.readCSV(pathData)
	if err != nil {
		return
	}

	err = es.savePlaces(places)
	if err != nil {
		return
	}
}

func (es *ElasticStore) CreateIndexWithMapping(index, pathStruct string) error {
	ctx := context.Background()

	exists, err := es.Client.IndexExists(index).Do(ctx)
	if err != nil {
		log.Print("Error checking if index exists: ")
		return err
	}

	schemaBytes, err := os.ReadFile(pathStruct)
	if err != nil {
		return err
	}

	if !exists {
		createIndex, err := es.Client.CreateIndex(index).BodyString(string(schemaBytes)).Do(ctx)
		if err != nil {
			log.Fatalf("Error creating index: %s", err)
			return err
		}
		if !createIndex.Acknowledged {
			log.Println("CreateIndex was not acknowledged. Check that timeout value is correct.")
		}
	} else {
		log.Println("Index already exists.")
		es.Index = index
		return nil
	}

	settings := map[string]interface{}{
		"index": map[string]interface{}{
			"max_result_window": 20000,
		},
	}

	err = es.updateIndexSettings(index, settings)
	if err != nil {
		return err
	}

	log.Println("Index created!")
	es.Index = index
	return nil
}

func (es *ElasticStore) GetPlaces(limit, offset int) ([]types.Place, int, error) {
	ctx := context.Background()

	searchResult, err := es.Client.Search().
		Index(es.Index).
		Query(elastic.NewMatchAllQuery()).
		Size(limit).
		From(offset).
		Do(ctx)
	if err != nil {
		return nil, 0, err
	}

	var places []types.Place
	for _, hit := range searchResult.Hits.Hits {
		var place types.Place
		err = json.Unmarshal(hit.Source, &place)
		if err != nil {
			log.Printf("Error unmarshalling hit source: %s", err)
			continue
		}
		places = append(places, place)
	}

	count, err := es.Client.Count().Index(es.Index).Do(ctx)
	if err != nil {
		return nil, 0, err
	}

	return places, int(count), nil
}

func (es *ElasticStore) readCSV(filePath string) ([]types.Place, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var places []types.Place
	var mutex sync.Mutex
	var wg sync.WaitGroup

	for i, record := range records {
		if i == 0 {
			continue
		}

		wg.Add(1)
		go func(record []string) {
			defer wg.Done()
			longitude, _ := strconv.ParseFloat(record[4], 64)
			latitude, _ := strconv.ParseFloat(record[5], 64)

			place := types.Place{
				ID:      record[0],
				Name:    record[1],
				Address: record[2],
				Phone:   record[3],
				Location: types.GeoPoint{
					Lat: latitude,
					Lon: longitude,
				},
			}
			mutex.Lock()
			places = append(places, place)
			mutex.Unlock()
		}(record)
	}

	wg.Wait()
	return places, nil
}

func (es *ElasticStore) savePlaces(places []types.Place) error {
	ctx := context.Background()
	bulkRequest := es.Client.Bulk()

	for _, place := range places {
		req := elastic.NewBulkIndexRequest().Index(es.Index).Id(place.ID).Doc(place)
		bulkRequest = bulkRequest.Add(req)
	}

	bulkResponse, err := bulkRequest.Do(ctx)
	if err != nil {
		log.Fatalf("Error executing bulk request: %s", err)
		return err
	}

	if bulkResponse != nil {
		for _, item := range bulkResponse.Items {
			for _, op := range item {
				if op.Error != nil {
					log.Printf("Failed to execute operation: %s", op.Error.Reason)
				}
			}
		}
	}

	return nil
}

func (es *ElasticStore) updateIndexSettings(index string, settings map[string]interface{}) error {
	ctx := context.Background()

	_, err := es.Client.IndexPutSettings(index).BodyJson(settings).Do(ctx)
	if err != nil {
		log.Fatalf("Error updating index settings: %s", err)
		return err
	}

	log.Println("Index settings updated successfully!")
	return nil
}
