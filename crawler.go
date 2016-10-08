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

var visited = make(map[string]bool)

type Crawler struct {

    startUrl string

    host string

    regex *regexp.Regexp

    queue chan string
}

func (crawler *Crawler) run() {
    go func() { 
        crawler.queue <- crawler.startUrl
    }()

    for url := range crawler.queue {
        crawler.crawl(url)
    }
}

func (crawler *Crawler) crawl(currentUrl string) {
    response, error := http.Get(currentUrl)
    
    if error != nil {
        fmt.Println("Failed GET to: ", currentUrl, " error: ", error)
    } else {
        fmt.Println("Fetching: ", currentUrl)
        body, error := ioutil.ReadAll(response.Body)
        if error != nil {
            fmt.Println("Failed to read response body, error: ", error)
        } else {
            strBody := string(body)
            crawler.extractUrls(currentUrl, strBody)
        }

        response.Body.Close()
    }
}

func (crawler *Crawler) extractUrls(stringUrl, body string) {
    newUrls := crawler.regex.FindAllStringSubmatch(body, -1)
    currentUrl, _ := url.Parse(stringUrl)

    if newUrls != nil {
        parsedUrls := parseUrls(newUrls, currentUrl)

        for _, newUrl := range parsedUrls { 
            if shouldVisit(newUrl, crawler.host) {
                go func(url string) {
                    crawler.queue <- url
                } (newUrl.String())
            }
        }
    }
}

func shouldVisit(url *url.URL, startHost string) bool {
    newUrlHost := getNormalizedHost(url.Host)
    shouldVisit := newUrlHost == startHost && !visited[url.String()]
    visited[url.String()] = true
    return shouldVisit
}

func parseUrls(rawUrls [][]string, currentUrl *url.URL) []*url.URL {
    urls := make([]*url.URL, 0)

    for _, raw := range rawUrls {
        stringUrl := raw[1];
        newUrl, error := url.Parse(stringUrl)
        if error != nil {
            continue
        }

        newUrl = makeAbsolute(currentUrl, newUrl)
        if (isValidUrl(newUrl)) {
            urls = append(urls, newUrl)
        }
    }

    return urls
}

func isValidUrl(url *url.URL) bool {
    return url.Host != ""
}

func makeAbsolute(currentUrl, u *url.URL) *url.URL {
    if !u.IsAbs() {
        return currentUrl.ResolveReference(u)
    }

    return u
}

func getNormalizedHost(host string) string {
    parts := strings.Split(host, ".")
    return strings.Join(parts[len(parts) - 2:], ".")
}

func main() {
    startUrl := os.Args[1]
    regex := regexp.MustCompile("<a.*?href=\"([^\"]*)\".*?>")

    u, _ := url.Parse(startUrl)
    host := getNormalizedHost(u.Host)

    crawler := Crawler {
        startUrl,
        host,
        regex,
        make(chan string),
    };

    crawler.run()
}