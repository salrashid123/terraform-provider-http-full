package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var ()

func main() {

	caCert, err := ioutil.ReadFile("certs/CA_crt.pem")
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	clientCerts, err := tls.LoadX509KeyPair(
		"certs/client.crt",
		"certs/client.key",
	)

	tlsConfig := &tls.Config{
		ServerName:   "localhost",
		Certificates: []tls.Certificate{clientCerts},
		RootCAs:      caCertPool,
		InsecureSkipVerify: true,
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get("https://localhost:8081/get")
	if err != nil {
		log.Println(err)
		return
	}

	htmlData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("%v\n", resp.Status)
	fmt.Printf(string(htmlData))

	/// **************************

	values := map[string]string{"ff": "bb", "cc": "dd"}
	json_data, err := json.Marshal(values)
	if err != nil {
		log.Fatal(err)
	}

	resp, err = client.Post("https://localhost:8081/post", "application/json", bytes.NewBuffer(json_data))
	if err != nil {
		log.Println(err)
		return
	}

	htmlData, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("\n%v\n", resp.Status)
	fmt.Printf(string(htmlData))

}
