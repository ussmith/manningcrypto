package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/kelseyhightower/envconfig"
)

type (
	EnvVariables struct {
		Path string `envconfig:"FILE_PATH"`
	}

	Request struct {
		Text string `json:"plain_text"`
	}

	PostResponse struct {
		ID string `json:"id"`
	}

	GetResponse struct {
		Data string `json:"data"`
	}
)

var (
	pwpairs map[string]string
	environ EnvVariables
)

const (
	defaultPath   string = "./"
	pairsFileName string = "data.json"
)

func main() {

	pwpairs = make(map[string]string)

	err := envconfig.Process("SECRETS", &environ)

	if err != nil {
		log.Fatalf("Failed to load environment: %v", err)
	}

	if !strings.HasSuffix(environ.Path, "/") {
		environ.Path += "/"
	}

	loadFromFile()

	mux := http.NewServeMux()

	//http.HandleFunc("secretHandler/:id", secretGetter)
	mux.HandleFunc("/healthcheck", healthcheck)
	mux.HandleFunc("/", secretHandler)
	//mux.HandleFunc("/secretHandler/:id", secretGetter)

	go listenToExit()
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

func listenToExit() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	fmt.Println("awaiting signal")
	<-done
	writeToFile()
	log.Println("Back from export to File, trying to exit")
	os.Exit(0)
}

func loadFromFile() {
	log.Print("In loadFromFile")

	file := environ.Path + pairsFileName
	if _, err := os.Stat(file); err == nil {
		bytes, err := os.ReadFile(file)
		if err != nil {
			//Failed to read, have to start with an empty map
			//Not going to blow it up, because it would fail the same way the next time
			log.Printf("Failed to load the save file, skipping: %v", err)
		} else if os.IsNotExist(err) {
			log.Println("File doesn't exist, could be the first run")
			return
		}

		err = json.Unmarshal(bytes, &pwpairs)

		if err != nil {
			log.Println("Failed to unmarshal, truncating the file")
			//Data corruption? Truncate the file to clear the cache
			os.Truncate(file, 0)
		}
	}

	//We're either in memory or on disk, but not both
	_ = os.Truncate(file, 0)
}

func writeToFile() {
	log.Print("In exportToFile")

	b, err := json.Marshal(&pwpairs)

	if err != nil {
		log.Printf("Failed to marshal map into json, bailing: %v", err)
		return
	}

	file := environ.Path + pairsFileName
	err = os.WriteFile(file, b, 0644)

	if err != nil {
		log.Printf("Failed to write file %v", err)
	}
}
