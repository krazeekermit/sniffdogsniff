package webui

import (
	"net/http"

	"github.com/sniffdogsniff/core"
	"github.com/sniffdogsniff/logging"
)

const WEBUI = "webui"

func getVarOrDefault_GET(r *http.Request, varName, def string) string {
	if r.URL.Query().Has(varName) {
		return r.URL.Query().Get(varName)
	} else {
		return def
	}
}

type SdsWebServer struct {
	node        *core.LocalNode
	resultsView *resultsPageView
}

func InitSdsWebServer(node *core.LocalNode) SdsWebServer {
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
			logging.Errorf(WEBUI, err.Error())
		}
		title := r.FormValue("link_title")
		url := r.FormValue("link_url")
		description := r.FormValue("link_description")
		dataType := core.StrToDataType(r.FormValue("data_type"))
		logging.Infof(WEBUI, "inserti link %s", dataType)
		server.node.InsertSearchResult(core.NewSearchResult(title, url,
			core.ResultPropertiesMap{core.RP_DESCRIPTION: description}, dataType))
	}
	renderTemplate(w, "insert_link.html", nil)
}

func (server *SdsWebServer) invalidateLinkHandleFunc(w http.ResponseWriter, r *http.Request) {
	server.node.InvalidateSearchResult(core.B64UrlsafeStringToHash(r.URL.Query().Get("hash")))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (server *SdsWebServer) ServeWebUi(address string) {
	srvmux := http.NewServeMux()

	srvmux.Handle("/", http.FileServer(http.FS(staticDir())))
	srvmux.HandleFunc("/search", server.searchHandleFunc)
	srvmux.HandleFunc("/redirect", server.redirectHandleFunc)
	srvmux.HandleFunc("/insert_link", server.insertLinkHandleFunc)
	srvmux.HandleFunc("/invalidate", server.invalidateLinkHandleFunc)

	logging.Infof(WEBUI, "Web Server is listening on %s", address)
	http.ListenAndServe(address, srvmux)

}
