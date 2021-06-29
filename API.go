/*
package main

import (
	"encoding/csv"
	"fmt"
	"net/http"
)

var dt []*Data

func LoadData() {
	URL := "https://raw.githubusercontent.com/Diegolivia/GoBackendKnn/main/Data/D_NORM_100.csv"
	response, err := http.Get(URL)
	if err != nil {
		fmt.Println(err)
	}
	defer response.Body.Close()
	reader := csv.NewReader(response.Body)
	reader.Read()
	data, _ := reader.ReadAll()
	for _, v := range data {
		aux := toData(v)
		dt = append(dt, &aux)
	}
}

func main() {

}
*/