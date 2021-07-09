package main

import (
	"fmt"
	"log"
	"net/http"
)

func DoLoadData(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Request Received Load Data")

	fmt.Println("Loading Data Finished")
}

func DoMath(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Request Received Do Math")
}

func handleRequest() {
	http.HandleFunc("/Doload", DoLoadData)
	http.HandleFunc("/DoMath", DoMath)
	log.Fatal(http.ListenAndServe(":6969", nil))
}

func main() {
	handleRequest()
}
