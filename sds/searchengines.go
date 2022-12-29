package sds

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/gocolly/colly"
	"gitlab.com/sniffdogsniff/util/logging"
)

type SearchEngine struct {
	name                    string
	userAgent               string
	searchQueryUrl          string
	resultsContainerElement string
	resultContainerElement  string
	resultUrlElement        string
	resultUrlProperty       string
	resultTitleElement      string
	resultTitleProperty     string
}

func (se SearchEngine) DoSearch(query string) []SearchResult {
	searchResults := make([]SearchResult, 0)

	c := colly.NewCollector()
	c.UserAgent = se.userAgent

	c.OnError(func(_ *colly.Response, err error) {
		logging.LogError(err.Error())
	})

	c.OnResponse(func(r *colly.Response) {
		logging.LogTrace(fmt.Sprintf("Visited %s", r.Request.URL.String()))
	})

	c.OnHTML(se.resultsContainerElement, func(e *colly.HTMLElement) {
		e.ForEach(se.resultContainerElement, func(_ int, elContainer *colly.HTMLElement) {
			url := elContainer.ChildAttr(se.resultUrlElement, se.resultUrlProperty)
			title := ""
			if se.resultTitleProperty == "text" {
				title = elContainer.ChildText(se.resultUrlElement)
			} else {
				title = elContainer.ChildAttr(se.resultUrlElement, se.resultUrlProperty)
			}
			if validUrl(url) {
				result := NewSearchResult(title, url, "")
				result.ReHash()
				searchResults = append(searchResults, result)
			}
		})
	})

	searchUrlString := fmt.Sprintf(se.searchQueryUrl, query)
	logging.LogInfo("Receiving results from " + searchUrlString)

	c.Visit(searchUrlString)
	c.Wait()
	return searchResults
}

func validUrl(urlString string) bool {
	u, err := url.Parse(urlString)
	if err != nil {
		logging.LogTrace("Found invalid url " + urlString)
		return false
	}
	switch u.Scheme {
	case "http":
		return true
	case "https":
		return true
	}
	return false
}

func normalizeString(text string) string {
	const pattern = `(<\/?[a-zA-A]+?[^>]*\/?>)*`
	r := regexp.MustCompile(pattern)
	groups := r.FindAllString(text, -1)
	sort.Slice(groups, func(i, j int) bool {
		return len(groups[i]) > len(groups[j])
	})
	for _, group := range groups {
		if strings.TrimSpace(group) != "" {
			text = strings.ReplaceAll(text, group, "")
		}
	}
	endIndex := strings.Index(text, "  ")
	if endIndex > 1 {
		text = text[0:endIndex]
	}
	return text
}
