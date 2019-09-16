package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/handler"
	"github.com/doncarron/gotodo"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

const defaultPort = "5050"

var ipList = make(map[string]string)

func getIPList() map[string]string {
	if ipList == nil {
		// b := new(bytes.Buffer)
		f, err := os.OpenFile("IPs.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		defer f.Close()

		d := json.NewDecoder(f)

		// Decoding the serialized data
		err = d.Decode(&ipList)
		if err != nil {
			panic(err)
		}
	}

	return ipList
}

func writeIPList(list map[string]string) {
	b := new(bytes.Buffer)
	d := json.NewEncoder(b)
	// Decoding the serialized data
	d.Encode(list)

	var err = ioutil.WriteFile("IPs.json", b.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
}

// Middleware decodes the share session cookie and packs the session into context
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf(r.RemoteAddr)

			var list = getIPList()

			list[r.RemoteAddr] = time.Now().String()

			writeIPList(list)

			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	router := chi.NewRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Add CORS middleware around every request
	// See https://github.com/rs/cors for full option listing
	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		Debug:            true,
	}).Handler)

	router.Use(Middleware())

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Check against your desired domains here
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	router.Handle("/", handler.Playground("GraphQL playground", "/query"))
	router.Handle("/query", handler.GraphQL(gotodo.NewExecutableSchema(gotodo.Config{Resolvers: &gotodo.Resolver{}}), handler.WebsocketUpgrader(upgrader)))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
