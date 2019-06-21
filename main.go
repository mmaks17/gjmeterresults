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
	Jurl      string `json:"jurl"`
	Jlogin    string `json:"jlogin"`
	Jpassword string `json:"jpassword"`
	Gurl      string `json:"gurl"`
	Gkey      string `json:"gkey"`
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

var (
	bstr       []Binfo
	gurl       string
	config     conf
	configfile string
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
}

func getBingo() {
	jobName := "goldapple-stage-load-testing"
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

		fmt.Printf("Время старта:%s, Длительность:%d, %s, №%d, Порядок:%d\n", data.GetTimestamp(), data.GetDuration()/1000, data.GetResult(), build.Number, found)
		var b Binfo
		b.timestart = strconv.FormatInt(data.GetTimestamp().Unix(), 10)
		b.timeend = strconv.FormatInt(data.GetTimestamp().Unix()+data.GetDuration()/1000, 10)
		b.url = config.Gurl + "/query?db=comfy_load&q=SELECT%20count(responseTime)%20as%20Count%2C%20mean(responseTime)%20as%20Avg%2C%20min(responseTime)%20as%20Min%2C%20median(responseTime)%20as%20Median%2C%20percentile(responseTime%2C%2090)%20as%20%2290%25%22%2Cpercentile(responseTime%2C%2095)%20as%20%2295%25%22%2Cpercentile(responseTime%2C%2099)%20as%20%2299%25%22%2C%20max(responseTime)%20as%20Max%2C%20(sum(errorCount)%2Fcount(responseTime))%20as%20%22Error%20Rate%22%20FROM%20%22requestsRaw%22%20%20WHERE%20time%20%3E%3D%20" + b.timestart + "000ms%20and%20time%20%3C%3D%20" + b.timeend + "000ms%20GROUP%20BY%20requestName&epoch=ms"
		bstr = append(bstr, b)
		found++

		// if "SUCCESS" == data.GetResult() {
		// 	fmt.Printf("Время старта:%s, Длительность:%d, %s, №%d, Порядок:%d\n", data.GetTimestamp(), data.GetDuration()/1000, data.GetResult(), build.Number, found)
		// 	var b Binfo
		// 	b.timestart = strconv.FormatInt(data.GetTimestamp().Unix(), 10)
		// 	b.timeend = strconv.FormatInt(data.GetTimestamp().Unix()+data.GetDuration()/1000, 10)
		// 	bstr = append(bstr, b)
		// 	found++
		// }
	}
	fmt.Printf("Будет обработано  сборок: %d \n\r", len(bstr))
	if len(bstr) >= 2 {
		//		fmt.Printf("%+v\n", bstr[0])
		//		fmt.Printf("%+v\n", bstr[1])
	} else {
		log.Println("  недостаточно данных, обработано менее 2х сборок")
	}
}

func getGrafanainfo(url string) jTable {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// handle err
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
	minseries := 0
	if len(t[0].Results[0].Series) > len(t[1].Results[0].Series) {
		minseries = len(t[1].Results[0].Series)
	} else {
		minseries = len(t[0].Results[0].Series)
	}

	for i := 0; i < minseries; i++ {
		//if t[0].Results[0].Series[i].Tags.RequestName != "/" || t[1].Results[0].Series[i].Tags.RequestName != "/" {
		if t[0].Results[0].Series[i].Tags.RequestName != t[1].Results[0].Series[i].Tags.RequestName {
			continue
		}
		//fmt.Printf("new:%s old:%s\n", t[0].Results[0].Series[i].Tags.RequestName, t[1].Results[0].Series[i].Tags.RequestName)
		fmt.Println("Сравнимаваем результаты для пути '/'")

		fmt.Printf("new:%.0f old:%.0f\n", t[0].Results[0].Series[i].Values[i][2], t[1].Results[0].Series[i].Values[i][2])

		v1 := t[0].Results[0].Series[i].Values[i][2].(float64)
		v2 := t[1].Results[0].Series[i].Values[i][2].(float64)

		//fmt.Printf("%.0.f", in)
		if v1 > v2 {
			fmt.Println("AVG  стало хуже")
		} else {
			fmt.Println("AVG  стало лучше")
		}
		if t[0].Results[0].Series[i].Values[i][9].(float64) > t[1].Results[0].Series[i].Values[i][9].(float64) {
			fmt.Println("Error Rate  стало хуже")
		} else {
			fmt.Println("Error Rate  -  стало лучше")
		}

	}

}
