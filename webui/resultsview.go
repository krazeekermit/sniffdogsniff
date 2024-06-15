package webui

import (
	"fmt"
	"net/http"
	"net/url"
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

const RESULTS_TEMPLATE_FILE_NAME = "results_%s.html"

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
	if dataType == "" {
		dataType = "links"
	}

	//Avoid extra search actions
	if query != rpv.query {
		rpv.results = node.DoSearch(query)
		rpv.query = query
		logging.Debugf(WEBUI, "Found results %d", len(rpv.results))
	}

	if rpv.dataType != dataType {
		rpv.dataType = dataType
	}

	renderTemplate2(w, fmt.Sprintf(RESULTS_TEMPLATE_FILE_NAME, dataType), argsMap{
		"results":     filterSearchResults(rpv.results, urlFilter, dataType),
		"q":           rpv.query,
		"link_filter": urlFilter,
		"data_type":   dataType,
	})
}
