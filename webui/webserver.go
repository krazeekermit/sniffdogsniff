package main

import (
	"fmt"
	"net/http"
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
	results := server.node.DoSearch(r.URL.Query().Get("q"))
	renderTemplate(w, "results.html", results)
}

func (server *SdsWebServer) insertLinkHandleFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		title := r.Form.Get("link_title")
		url := r.Form.Get("link_url")
		description := r.Form.Get("link_description")
		server.node.InsertSearchResult(sds.NewSearchResult(title, url, description))
	}
	renderTemplate(w, "insert_link.html", nil)
}

func (server *SdsWebServer) ServeWebUi(address string) {
	srvmux := http.NewServeMux()

	srvmux.Handle("/", http.FileServer(http.FS(staticDir())))
	srvmux.HandleFunc("/search", server.searchHandleFunc)
	srvmux.HandleFunc("/insert_link", server.insertLinkHandleFunc)

	sds.LogInfo(fmt.Sprintf("Web Server is listening on %s", address))
	http.ListenAndServe(address, srvmux)

}
