package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "regexp"
    "sync"
)

type ArchiveSnapshot struct {
    Status           string `json:"status"`
    Available        bool   `json:"available"`
    URL              string `json:"url"`
    Timestamp        string `json:"timestamp"`
}

type WaybackResponse struct {
    URL              string                     `json:"url"`
    ArchivedSnapshots map[string]ArchiveSnapshot `json:"archived_snapshots"`
    Timestamp        string                     `json:"timestamp"`
}

func fetchURL(url string) (string, error) {
    resp, err := http.Get(url)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    return string(body), nil
}

func extractURLs(finalString string, domain string) []string {
    pattern := fmt.Sprintf(`http://web\.archive\.org/web/[0-9]{14}/https://%s/robots\.txt`, domain)
    re := regexp.MustCompile(pattern)
    urls := re.FindAllString(finalString, -1)
    return urls
}

func extractIFrameSrc(html string) string {
    re := regexp.MustCompile(`<iframe[^>]+id="playback"[^>]+src="([^"]+)"`)
    match := re.FindStringSubmatch(html)
    if len(match) > 1 {
        return match[1]
    }
    return ""
}

func main() {
    args := os.Args[1:]
    if len(args) != 1 {
        fmt.Println("Usage: gar <domain>")
        return
    }

    domain := args[0]
    var wg sync.WaitGroup
    results := make(chan WaybackResponse, 25)

    for year := 2001; year <= 2025; year++ {
        wg.Add(1)
        go getWaybackResponse(domain, year, &wg, results)
    }

    go func() {
        wg.Wait()
        close(results)
    }()

    var urls []string

    for result := range results {
        if snapshot, ok := result.ArchivedSnapshots["closest"]; ok {
            urls = append(urls, snapshot.URL)
        }
    }

    var finalResponse string

    for _, url := range urls {
        body, err := fetchURL(url)
        if err != nil {
            fmt.Printf("Error fetching URL %s: %v\n", url, err)
            continue
        }
        finalResponse += body
    }

    matches := extractURLs(finalResponse, domain)

    // Create a map to store unique URLs
    uniqueURLs := make(map[string]bool)
    for _, match := range matches {
        uniqueURLs[match] = true
    }

    // Convert unique URLs back to a slice
    var finalOutput []string
    for url := range uniqueURLs {
        finalOutput = append(finalOutput, url)
    }

  //  fmt.Println(finalOutput)

    var finalPaths []string

    // Make HTTP requests for each URL in finalOutput and print response bodies
    for _, url := range finalOutput {
        body, err := fetchURL(url)
        if err != nil {
            fmt.Printf("Error fetching URL %s: %v\n", url, err)
            continue
        }
 //       fmt.Printf("Response body for URL %s:\n%s\n", url, body)

        // Extract src attribute from iframe tag
        iframeSrc := extractIFrameSrc(body)
        if iframeSrc != "" {
   //         fmt.Println("Src attribute value of the iframe:", iframeSrc)
            // Make HTTP request to iframeSrc and save the response body
            iframeBody, err := fetchURL(iframeSrc)
            if err != nil {
                fmt.Printf("Error fetching URL %s: %v\n", iframeSrc, err)
                continue
            }
            // Append the regex matches to finalPaths
            re := regexp.MustCompile(`\/[^\s]+`)
            matches := re.FindAllString(iframeBody, -1)
            finalPaths = append(finalPaths, matches...)
            // Save the response body
            // Here, you can implement your logic to save the response body as per your requirement
     //       fmt.Printf("Response body for iframe URL %s:\n%s\n", iframeSrc, iframeBody)
        } else {
            //fmt.Println("No iframe with id=\"playback\" found or src attribute value is empty.")
              fmt.Println("No robots.txt found.")

        }
    }

    // Unique finalPaths
    uniqueFinalPaths := make(map[string]bool)
    for _, path := range finalPaths {
        uniqueFinalPaths[path] = true
    }

    // Convert unique finalPaths back to a slice
    var finalOutputPaths []string
    for path := range uniqueFinalPaths {
        finalOutputPaths = append(finalOutputPaths, path)
    }

   // fmt.Println("Unique final paths:")
    for _, path := range finalOutputPaths {
        fmt.Println(path)
    }
}

func getWaybackResponse(domain string, year int, wg *sync.WaitGroup, results chan<- WaybackResponse) {
    defer wg.Done()
    url := fmt.Sprintf("https://archive.org/wayback/available?url=%s/robots.txt&timestamp=%d0000000000", domain, year)
    body, err := fetchURL(url)
    if err != nil {
        fmt.Printf("Error fetching URL %s: %v\n", url, err)
        return
    }

    var response WaybackResponse
    if err := json.Unmarshal([]byte(body), &response); err != nil {
        fmt.Printf("Error unmarshalling JSON for URL %s: %v\n", url, err)
        return
    }

    results <- response
}
