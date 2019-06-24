package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/bndr/gojenkins"
)

type conf struct {
	Jurl        string  `json:"jurl"`
	Jtask       string  `json:"jtask"`
	Jlogin      string  `json:"jlogin"`
	Jpassword   string  `json:"jpassword"`
	Gurl        string  `json:"gurl"`
	Gkey        string  `json:"gkey"`
	Infelicity  float64 `json:"infelicity"`
	Defduration int64   `json:"defduration"`
}

func (c *conf) getConf() *conf {

	yamlFile, err := ioutil.ReadFile(configfile)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	jsonErr := json.Unmarshal(yamlFile, &config)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	return c
}

// my struct
type Binfo struct {
	timestart string
	timeend   string
	url       string
}
type jTable struct {
	Home         float64
	Category     float64
	Product      float64
	Addtocart    float64
	FullCheckout float64

	Results []struct {
		StatementID int `json:"statement_id"`
		Series      []struct {
			Name string `json:"name"`
			Tags struct {
				RequestName string `json:"requestName"`
			} `json:"tags"`
			Columns []string        `json:"columns"`
			Values  [][]interface{} `json:"values"`
		} `json:"series"`
	} `json:"results"`
}

func (j *jTable) calcRez() *jTable {
	for i := 0; i < len(j.Results[0].Series); i++ {
		//if t[0].Results[0].Series[i].Tags.RequestName != "/" || t[1].Results[0].Series[i].Tags.RequestName != "/" {

		//fmt.Printf("%+v\n", t[0].Results[0].Series[i].Values)
		//fmt.Printf("%+v\n", t[1].Results[0].Series[i].Values)

		if j.Results[0].Series[i].Tags.RequestName == "/" || j.Results[0].Series[i].Tags.RequestName == "/customer/section/load/" {
			j.Home += j.Results[0].Series[i].Values[0][2].(float64)
			//fmt.Printf("%s %f\n\n", j.Results[0].Series[i].Tags.RequestName, j.Results[0].Series[i].Values)
		}
		if j.Results[0].Series[i].Tags.RequestName == "/category" {
			j.Category += j.Results[0].Series[i].Values[0][2].(float64)
		}
		if j.Results[0].Series[i].Tags.RequestName == "/pdp" {
			j.Product += j.Results[0].Series[i].Values[0][2].(float64)
		}
		if j.Results[0].Series[i].Tags.RequestName == "/cart/add/" {
			j.Addtocart += j.Results[0].Series[i].Values[0][2].(float64)
		}
		if j.Results[0].Series[i].Tags.RequestName == "/customer/section/load/" ||
			j.Results[0].Series[i].Tags.RequestName == "/delivery" ||
			j.Results[0].Series[i].Tags.RequestName == "/estimate-shipping-methods" ||
			j.Results[0].Series[i].Tags.RequestName == "/GetValidAddress" ||
			j.Results[0].Series[i].Tags.RequestName == "/checkout/" ||
			j.Results[0].Series[i].Tags.RequestName == "/payment-information" ||
			j.Results[0].Series[i].Tags.RequestName == "/review/product/listAjax/" ||
			j.Results[0].Series[i].Tags.RequestName == "/shipping-information" ||
			j.Results[0].Series[i].Tags.RequestName == "/user-choice-gifts" {
			j.FullCheckout += j.Results[0].Series[i].Values[0][2].(float64)
		}

	}

	return j
}

var (
	bstr       []Binfo
	gurl       string
	config     conf
	configfile string

	infelicity float64
)

func init() {
	log.Print("gJmetesAnaliser version 0.01")
	// принимаем на входе флаг -configfile
	flag.StringVar(&configfile, "config", "", "configfile patch")
	//flag.IntVar(&delay, "configfile", 0, "delay qyery")
	flag.Parse()
	fmt.Println(configfile)

	// без него не запускаемся
	if configfile == "" {
		log.Print("-config is required")
		os.Exit(1)
	}

	config.getConf()
	infelicity = config.Infelicity
}

func getBingo() {
	jobName := config.Jtask
	jenkins := gojenkins.CreateJenkins(config.Jurl, config.Jlogin, config.Jpassword)
	_, err := jenkins.Init()
	if err != nil {

		panic("Something Went Wrong")
	}

	builds, err := jenkins.GetAllBuildIds(jobName)

	if err != nil {
		panic(err)
	}
	log.Printf("Всего сборок \"%s\": %d", jobName, len(builds))
	found := 0

	for _, build := range builds {
		if found == 2 {
			break
		}

		buildId := build.Number
		data, err := jenkins.GetBuild(jobName, buildId)

		if err != nil {
			panic(err)
		}

		if "SUCCESS" == data.GetResult() {
			fmt.Printf("Время старта:%s, Длительность:%d, %s, №%d, Порядок:%d\n", data.GetTimestamp(), data.GetDuration()/1000, data.GetResult(), build.Number, found)
			var b Binfo
			var duration int64
			if data.GetDuration()/1000 == 0 {
				duration = config.Defduration
			} else {
				duration = data.GetDuration() / 1000
			}
			b.timestart = strconv.FormatInt(data.GetTimestamp().Unix(), 10)
			b.timeend = strconv.FormatInt(data.GetTimestamp().Unix()+duration, 10)
			b.url = config.Gurl + "/query?db=comfy_load&q=SELECT%20count(responseTime)%20as%20Count%2C%20mean(responseTime)%20as%20Avg%2C%20min(responseTime)%20as%20Min%2C%20median(responseTime)%20as%20Median%2C%20percentile(responseTime%2C%2090)%20as%20%2290%25%22%2Cpercentile(responseTime%2C%2095)%20as%20%2295%25%22%2Cpercentile(responseTime%2C%2099)%20as%20%2299%25%22%2C%20max(responseTime)%20as%20Max%2C%20(sum(errorCount)%2Fcount(responseTime))%20as%20%22Error%20Rate%22%20FROM%20%22requestsRaw%22%20%20WHERE%20time%20%3E%3D%20" + b.timestart + "000ms%20and%20time%20%3C%3D%20" + b.timeend + "000ms%20GROUP%20BY%20requestName&epoch=ms"
			bstr = append(bstr, b)
			found++
		}
	}
	fmt.Printf("Будет обработано  сборок: %d \n\r", len(bstr))
	if len(bstr) < 2 {
		log.Panic("  недостаточно данных, обработано менее 2х сборок")
	}
}

func getGrafanainfo(url string) jTable {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic("not info from grafana")
	}
	req.Host = "g.oggy.co"
	req.Header.Set("Authorization", "Bearer "+config.Gkey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	//	responseString := string(responseData)

	var jtable jTable
	jsonErr := json.Unmarshal(responseData, &jtable)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}
	fmt.Printf(" Количество протестированных путей Series: %d \n", len(jtable.Results[0].Series))

	defer resp.Body.Close()
	return jtable
}

func main() {
	var t []jTable
	getBingo()

	for _, bs := range bstr {
		t = append(t, getGrafanainfo(bs.url))
	}
	for i := 0; i < len(t); i++ {
		t[i].calcRez()
		//fmt.Printf("#%d%+v\n", i, t[i].)
	}

	if (t[0].Home-t[1].Home)/t[1].Home*100 > infelicity {
		fmt.Println("home page SLOW")
		//fmt.Printf("new: %.2f; old: %.2f,  значения увеличились на: %.2f %%\n", t[0].Home, t[1].Home, (t[0].Home-t[1].Home)/t[1].Home*100)
	}
	if (t[0].Addtocart-t[1].Addtocart)/t[1].Addtocart*100 > infelicity {
		fmt.Println("Addtocart page  SLOW")
		//fmt.Printf("new: %.2f; old: %.2f,  значения увеличились на: %.2f %%\n", t[0].Addtocart, t[1].Addtocart, (t[0].Addtocart-t[1].Addtocart)/t[1].Addtocart*100)
	}

	if (t[0].Category-t[1].Category)/t[1].Category*100 > infelicity {
		fmt.Println("Category page  SLOW")
		//fmt.Printf("new: %.2f; old: %.2f,  значения увеличились на: %.2f %%\n", t[0].Category, t[1].Category, (t[0].Category-t[1].Category)/t[1].Category*100)
	}

	if (t[0].Product-t[1].Product)/t[1].Product*100 > infelicity {
		fmt.Println("Product page  SLOW")
		//fmt.Printf("new: %.2f; old: %.2f,  значения увеличились на: %.2f %%\n", t[0].Product, t[1].Product, (t[0].Product-t[1].Product)/t[1].Product*100)
	}

	if (t[0].FullCheckout-t[1].FullCheckout)/t[1].FullCheckout*100 > infelicity {
		fmt.Println("FullCheckout page  SLOW")
		//fmt.Printf("new: %.2f; old: %.2f,  значения увеличились на: %.2f %%\n", t[0].FullCheckout, t[1].FullCheckout, (t[0].FullCheckout-t[1].FullCheckout)/t[1].FullCheckout*100)
	}

	fmt.Println("//***********DEBUG_INFO****************************************************/")
	fmt.Printf("new: %.0f; old: %.0f,  значения Home  изменились  на: %.0f %%\n", t[0].Home, t[1].Home, (t[0].Home-t[1].Home)/t[1].Home*100)
	fmt.Printf("new: %.0f; old: %.0f,  значения Addtocart изменились на: %.0f %%\n", t[0].Addtocart, t[1].Addtocart, (t[0].Addtocart-t[1].Addtocart)/t[1].Addtocart*100)
	fmt.Printf("new: %.0f; old: %.0f,  значения Category изменились на: %.0f %%\n", t[0].Category, t[1].Category, (t[0].Category-t[1].Category)/t[1].Category*100)
	fmt.Printf("new: %.0f; old: %.0f,  значения Product изменились на: %.0f %%\n", t[0].Product, t[1].Product, (t[0].Product-t[1].Product)/t[1].Product*100)
	fmt.Printf("new: %.0f; old: %.0f,  значения FullCheckout изменились на: %.0f %%\n", t[0].FullCheckout, t[1].FullCheckout, (t[0].FullCheckout-t[1].FullCheckout)/t[1].FullCheckout*100)

}
