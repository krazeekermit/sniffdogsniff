package webui

import (
	"net/http"

	"gitlab.com/sniffdogsniff/sds"
	"gitlab.com/sniffdogsniff/util/logging"
)

type SdsWebServer struct {
	node *sds.LocalNode
}

func InitSdsWebServer(node *sds.LocalNode) SdsWebServer {
	return SdsWebServer{
		node: node,
	}
}

func (server *SdsWebServer) searchHandleFunc(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	urlFilter := r.URL.Query().Get("link_filter")
	results := filterSearchResults(server.node.DoSearch(query), urlFilter)
	renderTemplate(w, "results.html", results)
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
		server.node.InsertSearchResult(sds.NewSearchResult(title, url, description))
	}
	renderTemplate(w, "insert_link.html", nil)
}

func (server *SdsWebServer) ServeWebUi(address string) {
	srvmux := http.NewServeMux()

	srvmux.Handle("/", http.FileServer(http.FS(staticDir())))
	srvmux.HandleFunc("/search", server.searchHandleFunc)
	srvmux.HandleFunc("/redirect", server.redirectHandleFunc)
	srvmux.HandleFunc("/insert_link", server.insertLinkHandleFunc)

	logging.LogInfo("Web Server is listening on", address)
	http.ListenAndServe(address, srvmux)

}
