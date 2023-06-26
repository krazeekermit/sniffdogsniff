package webui

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/sniffdogsniff/sds"
	"github.com/sniffdogsniff/util/logging"
)

func getVarOrDefault_GET(r *http.Request, varName, def string) string {
	if r.URL.Query().Has(varName) {
		return r.URL.Query().Get(varName)
	} else {
		return def
	}
}

type searchActionStatus struct {
	results  []sds.SearchResult
	query    string
	dataType string
}

type SdsWebServer struct {
	node         *sds.LocalNode
	searchStatus *searchActionStatus
}

func InitSdsWebServer(node *sds.LocalNode) SdsWebServer {
	return SdsWebServer{
		node:         node,
		searchStatus: new(searchActionStatus),
	}
}

func (server *SdsWebServer) searchHandleFunc(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	urlFilter := getVarOrDefault_GET(r, "link_filter", RULE_ALL)

	dataType := getVarOrDefault_GET(r, "data_type", server.searchStatus.dataType)

	pageNum, err := strconv.Atoi(getVarOrDefault_GET(r, "page", "0"))
	if err != nil {
		pageNum = 0
	}

	//Avoid extra search actions
	if query != server.searchStatus.query || dataType != server.searchStatus.dataType {
		logging.LogTrace("status changed")
		server.searchStatus.results = filterSearchResults(server.node.DoSearch(query), urlFilter, dataType)
		server.searchStatus.query = query
		server.searchStatus.dataType = dataType
		pageNum = 0
	}

	npages := len(server.searchStatus.results) / MAX_RESULTS_PER_PAGE
	renderTemplate2(w, fmt.Sprintf("results_%s.html", dataType), argsMap{
		"results":       getResultsForPage(server.searchStatus.results, pageNum),
		"n_pages":       npages,
		"q":             server.searchStatus.query,
		"link_filter":   urlFilter,
		"data_type":     dataType,
		"page_num":      pageNum,
		"has_next_page": pageNum+1 < npages,
		"next_page":     pageNum + 1,
		"has_prev_page": pageNum > 0,
		"prev_page":     pageNum - 1,
	})
}

func (server *SdsWebServer) redirectHandleFunc(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	hash := r.URL.Query().Get("hash")
	server.node.UpdateResultScore(hash)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (server *SdsWebServer) insertLinkHandleFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			logging.LogError(err.Error())
		}
		title := r.FormValue("link_title")
		url := r.FormValue("link_url")
		description := r.FormValue("link_description")
		dataType := sds.StrToDataType(r.FormValue("data_type"))
		logging.LogTrace("inserti link", dataType)
		server.node.InsertSearchResult(sds.NewSearchResult(title, url,
			sds.ResultPropertiesMap{sds.RP_DESCRIPTION: description}, dataType))
	}
	renderTemplate(w, "insert_link.html", nil)
}

func (server *SdsWebServer) invalidateLinkHandleFunc(w http.ResponseWriter, r *http.Request) {
	server.node.InvalidateSearchResult(sds.B64UrlsafeStringToHash(r.URL.Query().Get("hash")))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (server *SdsWebServer) ServeWebUi(address string) {
	srvmux := http.NewServeMux()

	srvmux.Handle("/", http.FileServer(http.FS(staticDir())))
	srvmux.HandleFunc("/search", server.searchHandleFunc)
	srvmux.HandleFunc("/redirect", server.redirectHandleFunc)
	srvmux.HandleFunc("/insert_link", server.insertLinkHandleFunc)
	srvmux.HandleFunc("/invalidate", server.invalidateLinkHandleFunc)

	logging.LogInfo("Web Server is listening on", address)
	http.ListenAndServe(address, srvmux)

}
