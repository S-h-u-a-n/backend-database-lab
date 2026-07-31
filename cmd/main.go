package main

import (
	handlerutil "github.com/NYCU-SDC/summer/pkg/handler"
	"go.uber.org/zap"
	"net/http"
)

func healthz(w http.ResponseWriter, r *http.Request) {
	handlerutil.WriteJSONResponse(w, http.StatusOK, "ok")
}

func main(){
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)
	logger.Info("Starting backend service")

	err = databaseutil.MigrationUp("file://internal/database/migrations", "postgresql://postgres:password@localhost:5432/postgres?sslmode=disable", logger)
	if err != nil {
		logger.Fatal("Failed to run database migration", zap.Error(err))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	http.ListenAndServe(":8080", mux)
}