package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

var (
	pwpairs map[string]string
)

type Request struct {
	Text string `json:"plain_text"`
}

type PostResponse struct {
	ID string `json:"id"`
}

type GetResponse struct {
	Data string `json:"data"`
}

func main() {

	pwpairs = make(map[string]string)

	mux := http.NewServeMux()

	//http.HandleFunc("secretHandler/:id", secretGetter)
	mux.HandleFunc("/healthcheck", healthcheck)
	mux.HandleFunc("/", secretHandler)
	//mux.HandleFunc("/secretHandler/:id", secretGetter)

	http.ListenAndServe(":8080", mux)
}

func healthcheck(w http.ResponseWriter, r *http.Request) {

}

func secretHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		secretStorer(w, r)
	} else if r.Method == "GET" {
		secretGetter(w, r)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func secretStorer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("POST")

	fmt.Println("Read all")
	b, err := ioutil.ReadAll(r.Body)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Unable to extract the request body"))
		return
	}

	var localRequest Request
	fmt.Println("Unmarshal extract")
	err = json.Unmarshal(b, &localRequest)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Unable to Unmarshal the request body"))
		return
	}

	fmt.Printf("Write String, %s\n", localRequest.Text)
	d := []byte(localRequest.Text)
	h := md5.Sum(d)

	response := PostResponse{
		ID: hex.EncodeToString(h[:]),
	}

	bytes, err := json.Marshal(&response)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to marshal the response"))
		return
	}

	pwpairs[response.ID] = localRequest.Text
	w.Write(bytes)
}

func secretGetter(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GET")

	fmt.Printf("URL: %s\n", r.URL.Path)
	id := strings.TrimPrefix(r.URL.Path, "/")

	fmt.Printf("id = %s\n", id)
	v, ok := pwpairs[id]

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("The desired value was not found"))
	}

	gr := GetResponse{
		Data: v,
	}

	bytes, err := json.Marshal(gr)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to marshal response"))
	}

	w.Write(bytes)
}
