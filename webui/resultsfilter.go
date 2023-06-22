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
	if len(results) <= MAX_RESULTS_PER_PAGE {
		return results
	} else {
		firstIdx := MAX_RESULTS_PER_PAGE * page
		lastIdx := firstIdx + MAX_RESULTS_PER_PAGE
		if lastIdx > (len(results) - 1) {
			lastIdx = (len(results) - 1)
		}
		return results[firstIdx:lastIdx]
	}
}

func matchesUrlType(urlStr, urlType string) bool {
	url, err := url.Parse(urlStr)
	if err != nil {
		logging.LogTrace("URL filter -", err.Error())
		return true
	}
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

func filterSearchResults(results []sds.SearchResult, urlType, dataTypeStr string) []sds.SearchResult {
	filtered := make([]sds.SearchResult, 0)
	for _, sr := range results {
		logging.LogTrace(sr.Title, sr.DataType)
		if matchesUrlType(sr.Url, urlType) && sr.DataType == sds.StrToDataType(dataTypeStr) {
			filtered = append(filtered, sr)
		}
	}
	logging.LogTrace("FILTERED ", len(filtered), "results!!!!!!!!!!!!!!1")
	return filtered
}
