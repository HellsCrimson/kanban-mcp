package main

import (
	"flag"
	"log"
	"net/http"

	"kanban-mcp/internal/api"
	"kanban-mcp/internal/mcpserver"
	"kanban-mcp/internal/store"
	"kanban-mcp/internal/webui"
)

func main() {
	addr := flag.String("addr", ":8081", "listen address")
	dbPath := flag.String("db", "kanban.db", "path to the SQLite database file")
	flag.Parse()

	s, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer s.Close()

	mux := http.NewServeMux()
	api.Register(mux, s)
	mux.Handle("/mcp", mcpserver.NewHandler(s))
	mux.Handle("/", webui.Handler())

	log.Printf("kanban listening on http://localhost%s (UI: /, API: /api, MCP: /mcp)", *addr)
	if err := http.ListenAndServe(*addr, cors(mux)); err != nil {
		log.Fatal(err)
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Mcp-Session-Id, Last-Event-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
