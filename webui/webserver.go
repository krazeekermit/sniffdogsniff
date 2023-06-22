package webui

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/sniffdogsniff/sds"
	"github.com/sniffdogsniff/util"
	"github.com/sniffdogsniff/util/logging"
)

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
	urlFilter := RULE_ALL
	if r.URL.Query().Has("link_filter") {
		urlFilter = r.URL.Query().Get("link_filter")
	}

	dataType := r.URL.Query().Get("data_type")

	logging.LogTrace("---------_>>>>>>>>>>>>> WEBUI SEARCH", query, server.searchStatus.query, dataType)

	pageNum, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		pageNum = 0
	}

	//Avoid extra search actions
	if query != server.searchStatus.query || dataType != server.searchStatus.dataType {
		logging.LogTrace("status changed")
		server.searchStatus.results = filterSearchResults(server.node.DoSearch(query), urlFilter, dataType)
		server.searchStatus.query = query
		pageNum = 0
	}

	npages := len(server.searchStatus.results) / MAX_RESULTS_PER_PAGE
	renderTemplate2(w, fmt.Sprintf("results_%s.html", dataType), argsMap{
		"results":       getResultsForPage(server.searchStatus.results, pageNum),
		"n_pages":       npages,
		"q":             server.searchStatus.query,
		"link_filter":   urlFilter,
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
		server.node.InsertSearchResult(sds.NewSearchResult(title, url, description, dataType))
	}
	renderTemplate(w, "insert_link.html", nil)
}

func (server *SdsWebServer) invalidateLinkHandleFunc(w http.ResponseWriter, r *http.Request) {
	server.node.InvalidateSearchResult(util.B64UrlsafeStringToHash(r.URL.Query().Get("hash")))
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
