package main

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "net/http"
    "net/url"
    "os"
    "regexp"
    "strings"
)

var builder bytes.Buffer
var visited = make(map[string]bool)

type Crawler struct {

    startUrl string

    host string

    regex *regexp.Regexp

    assetRegex *regexp.Regexp
}

type Page struct {
    location string

    links []string

    assets []*url.URL
}

func (crawler *Crawler) run() string {
    buildStart()
    crawler.crawl(crawler.startUrl)
    buildEnd()

    return builder.String()
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
            urls := crawler.extractUrls(currentUrl, string(body))
            for _, url := range urls {
                crawler.crawl(url)
            }
        }

        response.Body.Close()
    }
}

func (crawler *Crawler) extractUrls(stringUrl, body string) []string {
    urls := make([]string, 0)

    newUrls := crawler.regex.FindAllStringSubmatch(body, -1)
    pageAssets := crawler.assetRegex.FindAllStringSubmatch(body, -1)

    currentUrl, _ := url.Parse(stringUrl)

    if newUrls != nil {
        parsedUrls := parseUrls(newUrls, currentUrl)

        for _, newUrl := range parsedUrls { 
            if shouldVisit(newUrl, crawler.host) {
                urls = append(urls, newUrl.String())
            }
        }

        parsedAssets := parseUrls(pageAssets, currentUrl)

        page := Page {
            stringUrl,
            urls,
            parsedAssets,
        };

        addPageToSiteMap(page)
    }

    return urls
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

// helper methods to build the sitemap
func buildStart() {
    builder.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
    builder.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
}

func buildEnd() {
    builder.WriteString("</urlset>")
}

func addPageToSiteMap(page Page) {
    builder.WriteString("  <url>\n")

    // add location
    builder.WriteString("    <loc>")
    builder.WriteString(page.location)
    builder.WriteString("</loc>\n")

    // add links
    builder.WriteString("    <links:links>\n")
    for _, link := range page.links {
        builder.WriteString("      <link>")
        builder.WriteString(link)
        builder.WriteString("</link>\n")
    }
    builder.WriteString("    </links:link>\n")

    //add assets
    builder.WriteString("    <assets>\n")
    for _, asset := range page.assets {
        builder.WriteString("      <asset>")
        builder.WriteString(asset.String())
        builder.WriteString("</asset>\n")
    }
    builder.WriteString("    </assets>\n")

    builder.WriteString("  </url>\n")
}

func main() {
    startUrl := os.Args[1]
    regex := regexp.MustCompile("<a.*?href=\"([^\"]*)\".*?>")
    assetRegex := regexp.MustCompile("<link.*?href=\"([^\"]*)\".*?>")

    u, _ := url.Parse(startUrl)
    host := getNormalizedHost(u.Host)

    crawler := Crawler {
        startUrl,
        host,
        regex,
        assetRegex,
    };

    fmt.Printf(crawler.run())
}