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


const Indent = "  "

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

func (crawler *Crawler) run() string {
    buildStart()
    crawler.crawl(crawler.startUrl)
    buildEnd()

    return builder.String()
}

func (crawler *Crawler) crawl(url *url.URL) {
    response, error := http.Get(url.String())
    
    if error != nil {
        logError("Failed GET to: " + url.String(), error)
        return
    }

    log("Fetching: " + url.String())

    body, error := ioutil.ReadAll(response.Body)
    if error != nil {
        logError("Failed to read response body", error)
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
                subUrls = append(subUrls, subUrl)
                visited[subUrl.String()] = true
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
        
        if isValidUrl(newUrl) {
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
    builder.WriteString(indent(1))
    builder.WriteString("<url>")
    builder.WriteString("\n")

    // add location
    buildLocationSection(page.location.String())
    // add links
    buildTagSection("links:link", "link", page.links)
    // add assets
    buildTagSection("assets", "asset", page.assets)

    builder.WriteString(indent(1))
    builder.WriteString("</url>")
    builder.WriteString("\n")
}

func buildTagSection(section, tag string, content []*url.URL) {
    builder.WriteString(indent(2))
    builder.WriteString("<" + section + ">")
    builder.WriteString("\n")

    for _, info := range content {
        builder.WriteString(indent(3))
        builder.WriteString("<" + tag + ">")
        builder.WriteString(info.String())
        builder.WriteString("</" + tag + ">")
        builder.WriteString("\n")
    }

    builder.WriteString(indent(2))
    builder.WriteString("</" + section + ">")
    builder.WriteString("\n")
}

func buildLocationSection(location string) {
    builder.WriteString(indent(2))
    builder.WriteString("<loc>")
    builder.WriteString(location)
    builder.WriteString("</loc>")
    builder.WriteString("\n")
}

func indent(count int) string {
    return strings.Repeat(Indent, count)
}

func log(message string) {
    fmt.Fprintln(os.Stderr, message)
}

func logError(message string, err error) {
    fmt.Fprintln(os.Stderr, message, "Error was:", err)
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