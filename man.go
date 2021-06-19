package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gocarina/gocsv"
)

type Data struct {
	C1 float64 `json:"AMBITO_INEI" csv:"AMBITO_INEI"`
	C2 float64 `json:"NACIONAL_EXTRANJERO" csv:"NACIONAL_EXTRANJERO"`
	C3 float64 `json:"EDAD" csv:"EDAD"`
	C4 float64 `json:"SEXO" csv:"SEXO"`
	C5 float64 `json:"PLAN_DE_SEGURO_SIS_EMPRENDEDOR" csv:"PLAN_DE_SEGURO_SIS EMPRENDEDOR"`
	C6 float64 `json:"PLAN_DE_SEGURO_SIS_GRATUITO" csv:"PLAN_DE_SEGURO_SIS GRATUITO"`
	C7 float64 `json:"PLAN_DE_SEGURO_SIS_INDEPENDIENTE" csv:"PLAN_DE_SEGURO_SIS INDEPENDIENTE"`
	C8 float64 `json:"PLAN_DE_SEGURO_SIS_MICROEMPRESA" csv:"PLAN_DE_SEGURO_SIS MICROEMPRESA"`
	C9 float64 `json:"PLAN_DE_SEGURO_SIS_PARA_TODOS" csv:"PLAN_DE_SEGURO_SIS PARA TODOS"`
}

type Standarizer struct {
	min float64
	max float64
}

type OriginalAge struct {
	pos int
	age float64
}

func GetHeader(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	aux := scanner.Text()
	head := strings.Split(aux, ",")
	return head
}

func GetBody(path string) []*Data {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	dt := []*Data{}
	if err := gocsv.UnmarshalFile(file, &dt); err != nil {
		panic(err)
	}
	return dt
}

//Calcular la distancia de target a train data
func CalcDist(tgt, trn Data) float64 {
	return math.Sqrt(math.Pow(tgt.C1-trn.C1, 2) + math.Pow(tgt.C2-trn.C2, 2) + math.Pow(tgt.C3-trn.C3, 2) + math.Pow(tgt.C4-trn.C4, 2) + math.Pow(tgt.C5-trn.C5, 2) + math.Pow(tgt.C6-trn.C6, 2) + math.Pow(tgt.C7-trn.C7, 2) + math.Pow(tgt.C8-trn.C8, 2) + math.Pow(tgt.C9-trn.C9, 2))
}

func WorkData(tgt Data, trn <-chan Data, res chan<- float64, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range trn {
		a := CalcDist(tgt, j)
		fmt.Println(a)
		//res <- a
	}
}

func StandarizeAge(stnd Standarizer, dt []*Data) {
	for v := range dt {
		*&dt[v].C3 = (*&dt[v].C3 - stnd.min) / (stnd.max - stnd.min)
	}
}

func StandarizeTarget(stnd Standarizer, dt *Data) {
	dt.C3 = (dt.C3 - stnd.min) / (stnd.max - stnd.min)
}

func knn(target Data, Tdata []*Data) {
	jobs := make(chan Data)
	res := make(chan float64, len(Tdata))
	wg := new(sync.WaitGroup)
	fmt.Println("Start Knn")
	for a := 0; a <= 3; a++ {
		wg.Add(1)
		go WorkData(target, jobs, res, wg)
	}
	for _, train := range Tdata {
		jobs <- *train
	}
	close(jobs)
	go func() {
		wg.Wait()
		close(res)
	}()
	for v := range res {
		fmt.Println(v)
	}
	fmt.Println("End Knn")
}

func LoadTarget(jso string) Data {
	var tar Data
	json.Unmarshal([]byte(jso), &tar)
	return tar
}

func HomePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the HomePage!")
	fmt.Println("Endpoint Hit: homePage")
}

func handleRequests() {
	http.HandleFunc("/", HomePage)
	log.Fatal(http.ListenAndServe(":10000", nil))
}

//TODO: ADD NEW COLUMN FOR AGE
//TODO: ADD WAY TO SWITCH COLUMNS ON THE FLY
//TODO: ADD CONCURRENCY FOR DATA PARSING
//TODO: ADD KNN EXECUTION
//TODO: ADD CRUD
//TODO: ADD NETWORK CONECTION
//TODO: ADD TARGET DATA FORMAT

func main() {
	file := "Data/D_NORMv2_100.csv"
	targetD := `{"AMBITO_INEI": 1,"NACIONAL_EXTRANJERO": 1,"EDAD": 60,"SEXO": 1,"PLAN_DE_SEGURO_SIS_EMPRENDEDOR": 1,"PLAN_DE_SEGURO_SIS_GRATUITO": 1,"PLAN_DE_SEGURO_SIS_INDEPENDIENTE": 0,"PLAN_DE_SEGURO_SIS_MICROEMPRESA": 0,"PLAN_DE_SEGURO_SIS_PARA_TODOS": 0}`
	header := GetHeader(file)
	fmt.Println(header)
	dt := []*Data{}

	fmt.Println("Data")
	dt = GetBody(file)

	stAge := Standarizer{min: 0, max: 120}
	StandarizeAge(stAge, dt)

	fmt.Println("Target")
	dt_tgt := LoadTarget(targetD)
	fmt.Println(dt_tgt)
	StandarizeTarget(stAge, &dt_tgt)
	fmt.Println(dt_tgt)
	knn(dt_tgt, dt)
}
