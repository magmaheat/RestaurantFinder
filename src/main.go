package main

import (
	"Go_Day03/src/db"
	"Go_Day03/src/handlers"
	"Go_Day03/src/token"
	"Go_Day03/src/types"
	"html/template"
	"log"
	"net/http"
)

//brew install go-task/tap/go-task

func main() {
	if len(token.MySigningKey) == 0 {
		log.Fatal("MY_SIGNING_KEY environment variable is not set")
	}

	client, err := db.NewElasticStore()
	if err != nil {
		log.Fatal(err)
		return
	}
	defer client.Client.Stop()

	err = client.CreateIndexWithMapping("places", "./src/templates/schema.json")
	if err != nil {
		log.Fatal(err)
		return
	}

	tmpl, err := handlers.LoadTemplate("./src/templates/template.html")
	if err != nil {
		log.Fatal(err)
		return
	}

	client.LoadData("./materials/data.csv")

	handleKit(client, tmpl)

	log.Println("Server started at :8888")
	if err = http.ListenAndServe(":8888", nil); err != nil {
		log.Fatal(err)
	}
}

func handleKit(client types.DataStore, tmpl *template.Template) {
	http.HandleFunc("/api/get_token", token.GetToken)

	protected := http.NewServeMux()
	protected.HandleFunc("/api/recommend", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleRecommendAPI(w, r, client)
	})

	http.Handle("/api/recommend", token.JwtMiddleware(protected))

	http.HandleFunc("/places", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleGetPlacesHTML(w, r, client, tmpl)
	})

	http.HandleFunc("/api/places/", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleGetPlacesAPI(w, r, client)
	})
}
