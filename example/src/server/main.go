package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	//"net/http/httputil"

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
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func gethandler(w http.ResponseWriter, r *http.Request) {

	val := r.Context().Value(contextKey("event")).(parsedData)

	fmt.Fprint(w, fmt.Sprintf("%s ok", val))
}

func posthandler(w http.ResponseWriter, r *http.Request) {
	val := r.Context().Value(contextKey("event")).(parsedData)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}
	log.Printf("Data val %s [%s]", val, string(body))

	fmt.Fprint(w, "ok")
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
