package main

import (
    "fmt"
    "io/ioutil"
    "net/http"
    "net/url"
    "os"
    "regexp"
    "strings"
)

type Crawler struct {

    baseUrl string

    regex *regexp.Regexp
}

func (crawler *Crawler) crawl(url string) {
    response, error := http.Get(url)
    
    if error != nil {
        fmt.Println("Failed GET to: ", url, " error: ", error)
    } else {
        body, error := ioutil.ReadAll(response.Body)
        if error != nil {
            fmt.Println("Failed to read response body, error: ", error)
        } else {
            strBody := string(body)
            crawler.extractUrls(url, strBody)
        }

        response.Body.Close()
    }
}

func (crawler *Crawler) extractUrls(currentUrl, body string) {
    newUrls := crawler.regex.FindAllStringSubmatch(body, -1)
    
    // TODO: create a proper filter
    baseUrl, _ := url.Parse(currentUrl)
    parts := strings.Split(baseUrl.Host, ".")
    baseUrlHost := parts[1] + "." + parts[2]
    
    u := ""
    if newUrls != nil {
        for _, z := range newUrls {
            u = z[1]
            ur, err := url.Parse(z[1])
            if err == nil && ur.Host == baseUrlHost {
                if ur.IsAbs() == true {
                    fmt.Println(u)
                } else if ur.IsAbs() == false {
                    fmt.Println(baseUrl.ResolveReference(ur).String())
                } else if strings.HasPrefix(u, "//") {
                    fmt.Println("http:" + u)
                } else if strings.HasPrefix(u, "/") {
                    fmt.Println(baseUrl.Host + u)
                } else {
                    fmt.Println(currentUrl + u)
                }
            }
        }
    }
}

func main() {
    startUrl := os.Args[1]
    // TODO: look at alternative regexes
    regex := regexp.MustCompile("(?s)<a[ t]+.*?href=\"(http.*?)\".*?>.*?</a>")

    crawler := Crawler {
        startUrl,
        regex,
    }

    crawler.crawl(startUrl)
}