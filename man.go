package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
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

type OrdData struct {
	pos int
	dt  *Data
}

type dtAux struct {
	dt  Data
	val float64
}

type ColValue struct {
	pos int
	val float64
}

type Resp struct {
	cond1 string
	cond2 string
}

type ByVal []dtAux

func (a ByVal) Len() int           { return len(a) }
func (a ByVal) Less(i, j int) bool { return a[i].val < a[j].val }
func (a ByVal) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

//work Target Data
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

func StandarizeTarget(stnd Standarizer, dt *Data) {
	dt.C3 = (dt.C3 - stnd.min) / (stnd.max - stnd.min)
}

//Dataload is NOT concurrent
func GetBodylocal(path string) []*Data {
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

func toData(s []string) Data {
	var dt Data
	slc := make([]float64, len(s))
	for i := 0; i < len(s); i += 1 {
		sAux, _ := strconv.ParseFloat(s[i], 64)
		slc[i] = float64(sAux)
	}
	dt.C1 = slc[0]
	dt.C2 = slc[1]
	dt.C3 = slc[2]
	dt.C4 = slc[3]
	dt.C5 = slc[4]
	dt.C6 = slc[5]
	dt.C7 = slc[6]
	dt.C8 = slc[7]
	dt.C9 = slc[8]
	return dt
}

func GetBodyNet(path string) []*Data {
	response, err := http.Get(path)
	if err != nil {
		fmt.Println(err)
	}
	defer response.Body.Close()
	reader := csv.NewReader(response.Body)
	reader.Read()
	data, _ := reader.ReadAll()
	dt := []*Data{}
	for _, v := range data {
		aux := toData(v)
		dt = append(dt, &aux)
	}
	return dt
}

//Estandarizar la edad a 0-1 por motivos de evaluacion para el knn
func StandarizeAge(stnd Standarizer, dt []*Data) {
	for v := range dt {
		*&dt[v].C3 = (*&dt[v].C3 - stnd.min) / (stnd.max - stnd.min)
	}
}

//DeEstandarizar la Edad para mostrarla por el API
func UnStandarizeAge(ogA []ColValue, dt []*Data) {
	for v := range dt {
		*&dt[ogA[v].pos].C3 = ogA[v].val
	}
}

//Calcular la distancia de target a train data utilizando distancia euclideana
func CalcDist(tgt, trn Data) float64 {
	return math.Sqrt(math.Pow(tgt.C1-trn.C1, 2) + math.Pow(tgt.C2-trn.C2, 2) + math.Pow(tgt.C3-trn.C3, 2) + math.Pow(tgt.C5-trn.C5, 2) + math.Pow(tgt.C6-trn.C6, 2) + math.Pow(tgt.C7-trn.C7, 2) + math.Pow(tgt.C8-trn.C8, 2) + math.Pow(tgt.C9-trn.C9, 2))
}

func tryAdd(arr []dtAux, dt dtAux, max int) []dtAux {
	arr = append(arr, dt)
	sort.Sort(ByVal(arr))
	return arr[:max]
}

//Evaluar la distancia de target a la data de training recibida del canal y devuelve solo las distancias menores
func WorkData(tgt Data, trn <-chan OrdData, res chan<- []dtAux, wg *sync.WaitGroup, max int) {
	count := 0
	var kval []dtAux
	for j := range trn {
		a := CalcDist(tgt, *j.dt)
		if count < max {
			kval = append(kval, dtAux{*j.dt, a})
			sort.Sort(ByVal(kval))
		} else {
			if a < kval[len(kval)-1].val {
				//fmt.Println("resorting")
				kval = tryAdd(kval, dtAux{*j.dt, a}, max)
			}
		}
		count++
	}
	//fmt.Println(kval)
	res <- kval
	fmt.Println("Work done")
	defer wg.Done()
}

//Calcular la distancia de los Knn usando canales
func knn(target Data, Tdata []*Data, k int) Resp {
	jobs := make(chan OrdData)
	res := make(chan []dtAux, 40)
	wg := new(sync.WaitGroup)

	var res_Tb []dtAux
	NumBuff := 6
	fmt.Println("Start Knn")
	fmt.Println(fmt.Sprint("K value: ", k))
	fmt.Println(fmt.Sprint("Process buffer: ", NumBuff))

	for a := 0; a <= 3; a++ {
		wg.Add(1)
		go WorkData(target, jobs, res, wg, NumBuff)
	}
	for it, train := range Tdata {
		jobs <- OrdData{it, train}
	}

	close(jobs)

	wg.Wait()
	close(res)
	fmt.Println("Done waiting, channel closed")

	for v := range res {
		res_Tb = append(res_Tb, v...)
	}
	sort.Sort(ByVal(res_Tb))
	fmt.Println("Done loading, sorted data is on main")

	//Comparator Female(0)/Male(1)
	comp := []int{0, 0}
	for _, v := range res_Tb[:k] {
		fmt.Println(v)
		if v.dt.C4 == 0 {
			comp[0]++
		} else {
			comp[1]++
		}
	}

	r := Resp{}
	r.cond1 = strconv.Itoa((comp[0] * 100) / k)
	r.cond2 = strconv.Itoa((comp[1] * 100) / k)
	fmt.Println("Prediction for Female: ", r.cond1, "%")
	fmt.Println("Prediction for Male: ", r.cond2, "%")
	fmt.Println("End Knn")
	return r
}

func LoadTarget(jso string) Data {
	var tar Data
	json.Unmarshal([]byte(jso), &tar)
	return tar
}
func createNewData(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Request Received Target Data ")
	reqBody, _ := ioutil.ReadAll(r.Body)
	convertir_a_cadena := string(reqBody)

	fmt.Println("Loading Target Data...")
	dt_tgt = LoadTarget(convertir_a_cadena)
	fmt.Println(dt_tgt)

	fmt.Println("Standarizing Target Age...")
	StandarizeTarget(stAge, &dt_tgt)
	fmt.Println(dt_tgt)
}

func runKnn(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Request Received Run KNN")
	num, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/knn/"))
	knresp := knn(dt_tgt, dt, num)
	fmt.Println(knresp)

	js := fmt.Sprintf("{\"Mujer\":%s, \"Hombre\":%s}", knresp.cond1, knresp.cond2)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(js))

}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Homepage Endpoint Hit")
}

func handleRequest() {
	http.HandleFunc("/", homePage)
	//post
	http.HandleFunc("/data", createNewData)
	//get
	http.HandleFunc("/knn/", runKnn)
	log.Fatal(http.ListenAndServe(":6969", nil))
}

//TODO: ADD TARGET DATA FORMAT
var dt []*Data
var stAge Standarizer
var dt_tgt Data

func main() {
	//targetD := `{"AMBITO_INEI": 1,"NACIONAL_EXTRANJERO": 1,"SEXO":1,"EDAD": 20,"PLAN_DE_SEGURO_SIS_EMPRENDEDOR": 0,"PLAN_DE_SEGURO_SIS_GRATUITO": 1,"PLAN_DE_SEGURO_SIS_INDEPENDIENTE": 0,"PLAN_DE_SEGURO_SIS_MICROEMPRESA": 0,"PLAN_DE_SEGURO_SIS_PARA_TODOS": 0}`
	URL := "https://raw.githubusercontent.com/Diegolivia/GoBackendKnn/main/Data/D_NORM_100.csv"
	fmt.Println("Loading Data...")
	dt = GetBodyNet(URL)
	fmt.Println("Standarize Data Age...")
	stAge = Standarizer{min: 0, max: 120}
	StandarizeAge(stAge, dt)
	fmt.Println("Starting Backend...")
	handleRequest()
}
