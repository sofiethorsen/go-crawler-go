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
    startUrl *url.URL
    host string
    urlRegex *regexp.Regexp
    assetRegex *regexp.Regexp
}

type Page struct {
    location *url.URL
    links []*url.URL
    assets []*url.URL
}

func (crawler *Crawler) crawl(url *url.URL) {
    response, error := http.Get(url.String())
    
    if error != nil {
        fmt.Println("Failed GET to:", url, "exiting. Error: was", error)
        return
    }

    fmt.Println("Fetching:", url)

    body, error := ioutil.ReadAll(response.Body)
    if error != nil {
        fmt.Println("Failed to read response body, exiting. Error was: ", error)
        return
    }

    subUrls := crawler.getUrlsFromBody(url, string(body))
    for _, subUrl := range subUrls {
        crawler.crawl(subUrl)
    }

    response.Body.Close()
}

func (crawler *Crawler) getUrlsFromBody(parentUrl *url.URL, body string) []*url.URL {
    subUrls := make([]*url.URL, 0)
    subUrlMatches := getUnpackedMatches(crawler.urlRegex, body)
    assetMatches := getUnpackedMatches(crawler.assetRegex, body)

    if subUrlMatches != nil && assetMatches != nil {
        parsedAssets := parseUrls(assetMatches, parentUrl)
        parsedUrls := parseUrls(subUrlMatches, parentUrl)

        for _, subUrl := range parsedUrls { 
            if shouldVisit(subUrl, crawler.host) {
                visited[subUrl.String()] = true
                subUrls = append(subUrls, subUrl)
            }
        }

        page := Page {
            parentUrl,
            subUrls,
            parsedAssets,
        };

        addPageToSiteMap(page)
    }

    return subUrls
}

func getUnpackedMatches(regex *regexp.Regexp, content string) []string {
    result := regex.FindAllStringSubmatch(content, -1)

    unpacked := make([]string, 0)
    for _, match := range result {
        unpacked = append(unpacked, match[1])
    }

    return unpacked
}

func shouldVisit(url *url.URL, startHost string) bool {
    newUrlHost := getNormalizedHost(url.Host)
    return newUrlHost == startHost && !visited[url.String()]
}

func parseUrls(rawUrls []string, currentUrl *url.URL) []*url.URL {
    urls := make([]*url.URL, 0)

    for _, raw := range rawUrls {
        newUrl, _ := url.Parse(raw)
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
    builder.WriteString(page.location.String())
    builder.WriteString("</loc>\n")

    // add links
    builder.WriteString("    <links:links>\n")
    buildTagSection("link", page.links)
    builder.WriteString("    </links:link>\n")

    //add assets
    builder.WriteString("    <assets>\n")
    buildTagSection("asset", page.assets)
    builder.WriteString("    </assets>\n")

    builder.WriteString("  </url>\n")
}

func buildTagSection(tag string, content []*url.URL) {
    builder.WriteString("    <" + tag + ">\n")
    for _, info := range content {
        builder.WriteString("      <" + tag + ">")
        builder.WriteString(info.String())
        builder.WriteString("</" + tag + ">\n")
    }   
}

func (crawler *Crawler) run() string {
    buildStart()
    crawler.crawl(crawler.startUrl)
    buildEnd()

    return builder.String()
}

func main() {
    startUrl := os.Args[1]
    urlRegex := regexp.MustCompile("<a.*?href=\"([^\"]*)\".*?>")
    assetRegex := regexp.MustCompile("<link.*?href=\"([^\"]*)\".*?>")

    url, _ := url.Parse(startUrl)
    host := getNormalizedHost(url.Host)

    crawler := Crawler {
        url,
        host,
        urlRegex,
        assetRegex,
    };

    siteMap := crawler.run()
    fmt.Printf("\n" + siteMap)
}