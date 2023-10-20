package webui

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sniffdogsniff/core"
	"github.com/sniffdogsniff/logging"
)

const (
	RULE_ALL      string = "all"
	RULE_ONION    string = "onion"
	RULE_I2P      string = "i2p"
	RULE_CLEARNET string = "clearnet"
)

const MAX_RESULTS_PER_PAGE int = 12

const RESULTS_TEMPLATE_FILE_NAME = "results_%s.html"

func getResultsForPage(results []core.SearchResult, page int) []core.SearchResult {
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
		logging.Debugf(WEBUI, "URL filter - %s", err.Error())
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

func filterSearchResults(results []core.SearchResult, urlType, dataTypeStr string) []core.SearchResult {
	filtered := make([]core.SearchResult, 0)
	for _, sr := range results {
		if matchesUrlType(sr.Url, urlType) && sr.DataType == core.StrToDataType(dataTypeStr) {
			filtered = append(filtered, sr)
		}
	}
	return filtered
}

type resultsPageView struct {
	results  []core.SearchResult
	query    string
	dataType string
}

func (rpv *resultsPageView) handle(w http.ResponseWriter, r *http.Request, node *core.LocalNode) {
	query := r.URL.Query().Get("q")
	urlFilter := getVarOrDefault_GET(r, "link_filter", RULE_ALL)

	dataType := getVarOrDefault_GET(r, "data_type", rpv.dataType)

	pageNum, err := strconv.Atoi(getVarOrDefault_GET(r, "page", "0"))
	if err != nil {
		pageNum = 0
	}

	//Avoid extra search actions
	if query != rpv.query || dataType != rpv.dataType {
		rpv.results = filterSearchResults(node.DoSearch(query), urlFilter, dataType)
		rpv.query = query
		rpv.dataType = dataType
		pageNum = 0
	}

	npages := len(rpv.results) / MAX_RESULTS_PER_PAGE
	renderTemplate2(w, fmt.Sprintf(RESULTS_TEMPLATE_FILE_NAME, dataType), argsMap{
		"results":       getResultsForPage(rpv.results, pageNum),
		"n_pages":       npages,
		"q":             rpv.query,
		"link_filter":   urlFilter,
		"data_type":     dataType,
		"page_num":      pageNum,
		"has_next_page": pageNum+1 < npages,
		"next_page":     pageNum + 1,
		"has_prev_page": pageNum > 0,
		"prev_page":     pageNum - 1,
	})
}
