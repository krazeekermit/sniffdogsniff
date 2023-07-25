package webui

import (
	"net/http"

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

type SdsWebServer struct {
	node        *sds.LocalNode
	resultsView *resultsPageView
}

func InitSdsWebServer(node *sds.LocalNode) SdsWebServer {
	return SdsWebServer{
		node:        node,
		resultsView: new(resultsPageView),
	}
}

func (server *SdsWebServer) searchHandleFunc(w http.ResponseWriter, r *http.Request) {
	server.resultsView.handle(w, r, server.node)
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
