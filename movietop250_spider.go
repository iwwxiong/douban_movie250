package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// 爬去豆瓣电影TO50信息
// https://movie.douban.com/top250

type movieInfo struct {
	Top     int     //排名
	Href    string  //地址
	Title   string  //名称
	Img     string  //封面
	Star    float64 //评分
	Graders int     //评分人数
}
type byMovieInfo []*movieInfo

func (m byMovieInfo) Len() int               { return len(m) }
func (m byMovieInfo) Less(i int, j int) bool { return m[i].Top < m[j].Top }
func (m byMovieInfo) Swap(i int, j int)      { m[i], m[j] = m[j], m[i] }

const templ = `
排名: {{.Top}}
名称: {{.Title}}
评分: {{.Star}}
评分人数: {{.Graders}}
地址: {{.Href}}
封面: {{.Img}}
-----------------------------------------------`

var movieTemplate = template.Must(template.New("movielist").Parse(templ))

var indexURL = "https://movie.douban.com/top250"
var urlChannel = make(chan string, 200)
var userAgent = [...]string{"Mozilla/5.0 (compatible, MSIE 10.0, Windows NT, DigExt)",
	"Mozilla/4.0 (compatible, MSIE 7.0, Windojws NT 5.1, 360SE)",
	"Mozilla/4.0 (compatible, MSIE 8.0, Windows NT 6.0, Trident/4.0)",
	"Mozilla/5.0 (compatible, MSIE 9.0, Windows NT 6.1, Trident/5.0,",
	"Opera/9.80 (Windows NT 6.1, U, en) Presto/2.8.131 Version/11.11",
	"Mozilla/4.0 (compatible, MSIE 7.0, Windows NT 5.1, TencentTraveler 4.0)",
	"Mozilla/5.0 (Windows, U, Windows NT 6.1, en-us) AppleWebKit/534.50 (KHTML, like Gecko) Version/5.1 Safari/534.50",
	"Mozilla/5.0 (Macintosh, Intel Mac OS X 10_7_0) AppleWebKit/535.11 (KHTML, like Gecko) Chrome/17.0.963.56 Safari/535.11",
	"Mozilla/5.0 (Macintosh, U, Intel Mac OS X 10_6_8, en-us) AppleWebKit/534.50 (KHTML, like Gecko) Version/5.1 Safari/534.50",
	"Mozilla/5.0 (Linux, U, Android 3.0, en-us, Xoom Build/HRI39) AppleWebKit/534.13 (KHTML, like Gecko) Version/4.0 Safari/534.13",
	"Mozilla/5.0 (iPad, U, CPU OS 4_3_3 like Mac OS X, en-us) AppleWebKit/533.17.9 (KHTML, like Gecko) Version/5.0.2 Mobile/8J2 Safari/6533.18.5",
	"Mozilla/4.0 (compatible, MSIE 7.0, Windows NT 5.1, Trident/4.0, SE 2.X MetaSr 1.0, SE 2.X MetaSr 1.0, .NET CLR 2.0.50727, SE 2.X MetaSr 1.0)",
	"Mozilla/5.0 (iPhone, U, CPU iPhone OS 4_3_3 like Mac OS X, en-us) AppleWebKit/533.17.9 (KHTML, like Gecko) Version/5.0.2 Mobile/8J2 Safari/6533.18.5",
	"MQQBrowser/26 Mozilla/5.0 (Linux, U, Android 2.3.7, zh-cn, MB200 Build/GRJ22, CyanogenMod-7) AppleWebKit/533.1 (KHTML, like Gecko) Version/4.0 Mobile Safari/533.1"}
var r = rand.New(rand.NewSource(time.Now().UnixNano()))
var movieChan = make(chan movieInfo)

func getRandomUserAgent() string {
	return userAgent[r.Intn(len(userAgent))]
}

func parseHTML(html string) {
	lielements := regexp.MustCompile(`(?s)<div class="item">(.*?)</div>.+?</li>`).FindAllString(html, -1)
	re := regexp.MustCompile(`(?s).+?<em class="">(?P<top>\d+?)</em>.+?<a href="(?P<href>.*?)">.+?<img.+?alt="(?P<title>.*?)" src="(?P<img>.*?)" class="">.+?property="v:average">(?P<star>.*?)</span>.+?<span>(?P<graders>\d+)人评价</span>`)
	for _, li := range lielements {
		top := re.FindAllStringSubmatch(li, -1)[0]
		t, _ := strconv.Atoi(top[1])
		s, _ := strconv.ParseFloat(top[5], 64)
		g, _ := strconv.Atoi(top[6])
		mv := movieInfo{t, top[2], top[3], top[4], s, g}
		movieChan <- mv
	}
}

func spider(url string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("[E]", r)
		}
	}()

	// fmt.Printf("start request url: %s.\n", url)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", getRandomUserAgent())
	client := http.DefaultClient
	res, e := client.Do(req)
	if e != nil {
		fmt.Printf("Get 请求%s返回错误：%s", url, e)
		return
	}
	if res.StatusCode == 200 {
		body := res.Body
		defer body.Close()
		bodyByte, _ := ioutil.ReadAll(body)
		resStr := string(bodyByte)
		parseHTML(resStr)
	}
}

func main() {
	fmt.Printf("start crawl douban top250 movie info.\n")
	for p := 0; p < 10; p++ {
		url := "https://movie.douban.com/top250?start=" + strconv.Itoa(p*25)
		go spider(url)
	}
	result := make([]*movieInfo, 0, 250)
	for i := 0; i < 250; i++ {
		select {
		case x := <-movieChan:
			result = append(result, &x)
		}
	}
	sort.Sort(byMovieInfo(result)) // 按照排名重新排序
	for _, data := range result {
		if err := movieTemplate.Execute(os.Stdout, *data); err != nil {
			log.Fatal(err)
		}
	}
}
