package webui

import (
	"net/url"
	"strings"

	"github.com/sniffdogsniff/sds"
	"github.com/sniffdogsniff/util/logging"
)

const RULE_ALL string = "all"

const (
	RULE_ONION    string = "onion"
	RULE_I2P      string = "i2p"
	RULE_CLEARNET string = "clearnet"
)

const MAX_RESULTS_PER_PAGE int = 10

func getResultsForPage(results []sds.SearchResult, page int) []sds.SearchResult {
	firstIdx := MAX_RESULTS_PER_PAGE * page
	lastIdx := firstIdx + MAX_RESULTS_PER_PAGE
	if lastIdx > (len(results) - 1) {
		lastIdx = (len(results) - 1)
	}
	return results[firstIdx:lastIdx]
}

func matchesUrlType(url *url.URL, urlType string) bool {
	comps := strings.Split(url.Hostname(), ".")
	domain := comps[len(comps)-1]
	// logging.LogTrace("domain:", domain)
	switch urlType {
	case RULE_ALL:
		return true
	case RULE_CLEARNET:
		return domain != RULE_ONION && domain != RULE_I2P
	case RULE_ONION:
		return domain == RULE_ONION
	case RULE_I2P:
		return domain == RULE_I2P
	default:
		return true
	}
}

func filterSearchResults(results []sds.SearchResult, urlType string) []sds.SearchResult {
	if urlType == RULE_ALL {
		return results
	} else {
		filtered := make([]sds.SearchResult, 0)
		for _, sr := range results {
			url, err := url.Parse(sr.Url)
			if err != nil {
				logging.LogTrace("URL filter -", err.Error())
				continue
			}
			if matchesUrlType(url, urlType) {
				filtered = append(filtered, sr)
			}
		}
		return filtered
	}
}
