package main

import (
	handlerutil "github.com/NYCU-SDC/summer/pkg/handler"
	"net/http"
)

func healthz(w http.ResponseWriter, r *http.Request) {
	handlerutil.WriteJSONResponse(w, http.StatusOK, "ok")
}

func main(){
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	http.ListenAndServe(":8080", mux)
}