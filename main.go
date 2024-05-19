package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"regexp"
	"net/url"
)

var m sync.Map

type Fetcher interface {
	// Fetch returns the body of URL and a slice of URLs found on that page.
	Fetch(url string) (body string, urls []string, err error)
}

// realFetcher implements the Fetcher interface.
type realFetcher struct{}

func (f *realFetcher) Fetch(url string) (string, []string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("fetch failed with status: %s", resp.Status)
	}

	links := extractLinks(resp.Body)

	return "", links, nil
}

func getBaseURL(rawurl string) (string, error) {
	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host), nil
}

// Regular expression to find links in HTML
var linkRegex = regexp.MustCompile(`href=["'](https?://[^"']+)["']`)

func extractLinks(body io.Reader) []string {
    var links []string

    // Read HTML content from the reader
    htmlContent, err := ioutil.ReadAll(body)
    if err != nil {
        fmt.Println("Error reading HTML content:", err)
        return links
    }

    // Find all matches of links in the HTML content
    matches := linkRegex.FindAllStringSubmatch(string(htmlContent), -1)
    for _, match := range matches {
		baseURL, err := getBaseURL(match[1])
		if err != nil {
			continue
		}

        links = append(links, baseURL)
    }

    return links
}

// Crawl uses fetcher to recursively crawl pages starting with url, to a maximum of depth.
func Crawl(url string, depth int, fetcher Fetcher) {

	if depth <= 0 {
		return
	}

	// Don't fetch the same URL twice.
    _, ok := m.LoadOrStore(url, url)
    if ok {
        return
    }

	_, urls, err := fetcher.Fetch(url)
    if err != nil {
        fmt.Println(err)
        return
    }

	fmt.Printf("found: %s\n", url)

	var wg sync.WaitGroup
    defer wg.Wait()
	
	for _, u := range urls {
		wg.Add(1)
		go func(u string) {
            defer wg.Done()
            Crawl(u, depth-1, fetcher)
        }(u)
	}

	return
}

func main() {
	fmt.Println("crawling")
	Crawl("https://golang.org/", 100, &realFetcher{})
}