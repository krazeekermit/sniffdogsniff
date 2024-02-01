package core

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/sniffdogsniff/logging"
)

const CRAWLER = "crawler"

const NO_DESCRIPTION_AVAILABLE string = "No description available"

func validUrl(urlString string) bool {
	u, err := url.Parse(urlString)
	if err != nil {
		logging.Debugf(CRAWLER, "Found invalid url %s", urlString)
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

func extractUrlJson(jsonStr, property string) string {
	url := ""

	var m interface{}
	if json.Unmarshal([]byte(jsonStr), &m) == nil {
		switch v := m.(type) {
		case []interface{}:
			jsonMap, ok := v[0].(map[string]interface{})
			if ok {
				url = jsonMap[property].(string)
			}
		case map[string]interface{}:
			url = v[property].(string)
		}
	}
	return url
}

type SearchEngine struct {
	name                    string
	userAgent               string
	searchQueryUrl          string
	resultsContainerElement string
	resultContainerElement  string
	resultUrlElement        string
	resultUrlProperty       string
	resultUrlIsJson         bool
	resultUrlJsonProperty   string
	resultTitleElement      string
	resultTitleProperty     string
	providedDataType        ResultDataType
}

func (se SearchEngine) extractDescription(url string) string {
	c := colly.NewCollector()
	c.UserAgent = se.userAgent

	description := ""

	c.OnError(func(_ *colly.Response, err error) {
		logging.Errorf(CRAWLER, err.Error())
	})

	c.OnResponse(func(r *colly.Response) {
		logging.Debugf(CRAWLER, "Connecting to %s", r.Request.URL.String())
	})

	c.OnHTML("html", func(e *colly.HTMLElement) {
		description = e.ChildAttr("meta[name=\"description\"]", "content")
		if description == "" {
			description = e.ChildText("h1")
		}
	})

	c.Visit(url)
	c.Wait()

	if description == "" {
		description = NO_DESCRIPTION_AVAILABLE
	}
	return normalizeString(description)
}

func (se SearchEngine) DoSearch(ch chan []SearchResult, wg *sync.WaitGroup, query string) {
	// searchResults := make([]SearchResult, 0)

	// c := colly.NewCollector()
	// c.UserAgent = se.userAgent

	// c.OnError(func(_ *colly.Response, err error) {
	// 	logging.Errorf(CRAWLER, err.Error())
	// })

	// c.OnResponse(func(r *colly.Response) {
	// 	logging.Debugf(CRAWLER, "Visited %s", r.Request.URL.String())
	// })

	// c.OnHTML(se.resultsContainerElement, func(e *colly.HTMLElement) {
	// 	e.ForEach(se.resultContainerElement, func(_ int, elContainer *colly.HTMLElement) {
	// 		urlData := elContainer.ChildAttr(se.resultUrlElement, se.resultUrlProperty)

	// 		url := ""
	// 		if se.resultUrlIsJson {
	// 			url = extractUrlJson(urlData, se.resultUrlJsonProperty)
	// 		} else {
	// 			url = urlData
	// 		}

	// 		title := ""
	// 		if se.resultTitleProperty == "text" {
	// 			title = elContainer.ChildText(se.resultTitleElement)
	// 		} else {
	// 			title = elContainer.ChildAttr(se.resultTitleElement, se.resultTitleProperty)
	// 		}

	// 		if validUrl(url) {
	// 			desc := se.extractDescription(url)
	// 			result := NewSearchResult(title, url,
	// 				ResultPropertiesMap{RP_DESCRIPTION: desc}, se.providedDataType)
	// 			result.ReHash()
	// 			searchResults = append(searchResults, result)
	// 		}
	// 	})
	// })

	// searchUrlString := fmt.Sprintf(se.searchQueryUrl, query)
	// logging.Infof(CRAWLER, "Receiving results from %s", searchUrlString)

	// c.Visit(searchUrlString)
	// c.Wait()

	// ch <- searchResults
}

func DoParallelSearchOnExtEngines(engines map[string]SearchEngine, query string) []SearchResult {

	searchResults := make(map[Hash256]SearchResult)
	var lock sync.Mutex

	c := colly.NewCollector()
	for _, se := range engines {
		logging.Debugf("SEARCH ENGINE", se.name)
		c.UserAgent = se.userAgent

		c.OnError(func(_ *colly.Response, err error) {
			logging.Errorf(CRAWLER, err.Error())
		})

		c.OnResponse(func(r *colly.Response) {
			logging.Debugf(CRAWLER, "Visited %s", r.Request.URL.String())
		})

		c.OnHTML(se.resultsContainerElement, func(e *colly.HTMLElement) {
			_results := make([]SearchResult, 0)
			e.ForEach(se.resultContainerElement, func(_ int, elContainer *colly.HTMLElement) {
				urlData := elContainer.ChildAttr(se.resultUrlElement, se.resultUrlProperty)

				url := ""
				if se.resultUrlIsJson {
					url = extractUrlJson(urlData, se.resultUrlJsonProperty)
				} else {
					url = urlData
				}

				title := ""
				if se.resultTitleProperty == "text" {
					title = elContainer.ChildText(se.resultTitleElement)
				} else {
					title = elContainer.ChildAttr(se.resultTitleElement, se.resultTitleProperty)
				}

				if validUrl(url) {
					desc := se.extractDescription(url)
					result := NewSearchResult(title, url,
						ResultPropertiesMap{RP_DESCRIPTION: desc}, se.providedDataType)
					result.ReHash()

					_results = append(_results, result)
				}
			})
			lock.Lock()
			for i := 0; i < len(_results); i++ {
				sr := _results[i]
				searchResults[sr.ResultHash] = sr
			}
			lock.Unlock()
		})

		searchUrlString := fmt.Sprintf(se.searchQueryUrl, url.QueryEscape(query))
		logging.Infof(CRAWLER, "Receiving results from %s", searchUrlString)

		c.Visit(searchUrlString)
	}
	c.Wait()

	results := make([]SearchResult, 0)
	for _, sr := range searchResults {
		results = append(results, sr)
	}

	return results

}
