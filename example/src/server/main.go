package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"golang.org/x/net/http2"
)

var ()

const ()

type contextKey string

const contextEventKey contextKey = "event"

type parsedData struct {
	Msg string `json:"msg"`
}

func eventsMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		event := &parsedData{
			Msg: "foo",
		}

		ctx := context.WithValue(r.Context(), contextEventKey, *event)

		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		// theres got to be a better way here....
		rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
		rdr2 := ioutil.NopCloser(bytes.NewBuffer(buf))

		switch r.Header.Get("content-type") {
		case "application/json":
			log.Println("request  JSON::")
			// note, we are just parsing {key1=value1, key2=value2} json structure here
			// request_body = jsonencode({foo = "bar",bar = "bar"})
			jsonMap := make(map[string](string))
			err := json.Unmarshal(buf, &jsonMap)
			if err != nil {
				http.Error(w, "Error unmarshalling JSON", http.StatusBadRequest)
				return
			}
			log.Printf("INFO: jsonMap, %s", jsonMap)
		case "application/x-www-form-urlencoded":
			r.Body = rdr1
			r.ParseForm()
			log.Println("request.Form kv pairs: ")
			for key, value := range r.Form {
				log.Printf("Key:%s, Value:%s", key, value)
			}
		default:
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Printf("Error reading body: %v", err)
				http.Error(w, "can't read body", http.StatusBadRequest)
				return
			}
			log.Printf("Middleware Raw content: %s", body)

		}
		r.Body = rdr2
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func gethandler(w http.ResponseWriter, r *http.Request) {
	val := r.Context().Value(contextKey("event")).(parsedData)
	fmt.Fprintf(w, "%s ok", val)
}

func posthandler(w http.ResponseWriter, r *http.Request) {
	val := r.Context().Value(contextKey("event")).(parsedData)
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}
	log.Printf("Raw content: %s", body)
	fmt.Fprintf(w, "%s ok", val)
}

func main() {

	router := mux.NewRouter()
	router.Methods(http.MethodGet).Path("/get").HandlerFunc(gethandler)
	router.Methods(http.MethodPost).Path("/post").HandlerFunc(posthandler)

	clientCaCert, err := ioutil.ReadFile("certs/CA_crt.pem")
	clientCaCertPool := x509.NewCertPool()
	clientCaCertPool.AppendCertsFromPEM(clientCaCert)

	tlsConfig := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientCaCertPool,
	}

	var server *http.Server
	server = &http.Server{
		Addr:      ":8081",
		Handler:   eventsMiddleware(router),
		TLSConfig: tlsConfig,
	}
	http2.ConfigureServer(server, &http2.Server{})
	fmt.Println("Starting Server..")
	err = server.ListenAndServeTLS("certs/server.crt", "certs/server.key")
	fmt.Printf("Unable to start Server %v", err)

}
