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

const MAX_CONCURRENT_CRAWLERS = 5

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

func extractMetadata(userAgent, url string) (string, string, error) {
	c := colly.NewCollector()
	//c.UserAgent = userAgent

	description := ""
	title := ""
	var gerr error

	c.OnError(func(_ *colly.Response, err error) {
		gerr = err
	})

	c.OnResponse(func(r *colly.Response) {
		logging.Debugf(CRAWLER, "Connecting to %s", r.Request.URL.String())
	})

	c.OnHTML("title", func(h *colly.HTMLElement) {
		title = h.Text
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
	return title, normalizeString(description), gerr
}

type UrlQueue struct {
	indexes []string
}

func NewUrlQueue() UrlQueue {
	return UrlQueue{
		indexes: make([]string, 0),
	}
}

func (d *UrlQueue) push(i string) {
	d.indexes = append(d.indexes, i)
}

func (d *UrlQueue) popFirst() string {
	conn := d.indexes[0]
	d.indexes = d.indexes[1:]
	return conn
}

func (d *UrlQueue) isEmpty() bool {
	return len(d.indexes) == 0
}

func (d *UrlQueue) count() int {
	return len(d.indexes)
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

type Crawler struct {
	cond        *sync.Cond
	takeover    bool
	engines     map[string]SearchEngine
	callback    func(SearchResult)
	queue       UrlQueue
	probed      map[Hash256]SearchResult
	probedMutex *sync.Mutex
}

func NewCrawler(engines map[string]SearchEngine) *Crawler {
	return &Crawler{
		cond:        sync.NewCond(&sync.Mutex{}),
		engines:     engines,
		callback:    nil,
		queue:       NewUrlQueue(),
		probed:      map[Hash256]SearchResult{},
		probedMutex: &sync.Mutex{},
	}
}

func (crawler *Crawler) SetUpdateCallback(callback func(SearchResult)) {
	crawler.callback = callback
}

func (crawler *Crawler) RunTask() {
	c := colly.NewCollector()
	c.OnError(func(r *colly.Response, err error) {
		logging.Errorf(CRAWLER, "%s: %s", r.Request.URL.String(), err.Error())
	})

	c.OnResponse(func(r *colly.Response) {
		logging.Debugf(CRAWLER, "Connecting %s", r.Request.URL.String())
	})

	c.OnHTML("body", func(e *colly.HTMLElement) {
		e.ForEach("a", func(_ int, h *colly.HTMLElement) {
			href := h.Attr("href")
			title, desc, err := extractMetadata("", href)
			if err != nil {
				return
			}

			logging.Infof(CRAWLER, "New result found %s: %s", title, href)

			result := NewSearchResult(title, href,
				ResultPropertiesMap{RP_DESCRIPTION: normalizeString(desc)}, LINK_DATA_TYPE)
			result.ReHash()

			crawler.probedMutex.Lock()
			if _, ok := crawler.probed[result.ResultHash]; !ok {
				crawler.probed[result.ResultHash] = result
				crawler.queue.push(href)
			}
			crawler.probedMutex.Unlock()
		})
	})

	for {
		crawler.cond.L.Lock()
		for crawler.queue.isEmpty() || crawler.takeover {
			crawler.cond.Wait()
		}

		logging.Infof(CRAWLER, "Seeded with %d results", crawler.queue.count())

		roundSeeds := make([]string, MAX_CONCURRENT_CRAWLERS)
		for i := 0; i < MAX_CONCURRENT_CRAWLERS; i++ {
			roundSeeds[i] = crawler.queue.popFirst()
		}
		crawler.cond.L.Unlock()

		for i := 0; i < MAX_CONCURRENT_CRAWLERS; i++ {
			c.Visit(roundSeeds[i])
		}

		c.Wait()
	}
}

func (crawler *Crawler) Seed(seeds ...string) {
	for i := 0; i < len(seeds); i++ {
		crawler.queue.push(seeds[i])
	}
}

func (crawler *Crawler) ResultsToPublish() []SearchResult {
	results := make([]SearchResult, 0)
	for _, sr := range crawler.probed {
		results = append(results, sr)
	}
	return results
}

func (crawler *Crawler) DoSearch(query string) []SearchResult {
	crawler.cond.L.Lock()
	crawler.takeover = true
	crawler.cond.L.Unlock()

	searchResults := make(map[Hash256]SearchResult)
	var lock sync.Mutex

	c := colly.NewCollector()
	for _, se := range crawler.engines {
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
		crawler.queue.push(sr.Url)
		results = append(results, sr)
	}

	crawler.cond.L.Lock()
	crawler.takeover = false
	crawler.cond.Broadcast()
	crawler.cond.L.Unlock()

	return results

}
