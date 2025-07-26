package controller

import (
	"net/http"
	"opsicle/internal/common"

	"github.com/gorilla/mux"
)

func registerSessionRoutesV1(router *mux.Router, serviceLogs chan<- common.ServiceLog) {
	router.HandleFunc("", createSessionHandlerV1).Methods(http.MethodPost)
	router.HandleFunc("", getSessionHandlerV1).Methods(http.MethodGet)
	router.HandleFunc("", stopSessionHandlerV1).Methods(http.MethodDelete)
}

func createSessionHandlerV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelInfo, "this endpoint logs a session in")
	w.Write([]byte("create session"))
}

func getSessionHandlerV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelInfo, "this endpoint retrieves a user's session information")
	w.Write([]byte("get session info"))
}

func stopSessionHandlerV1(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(common.HttpContextLogger).(common.HttpRequestLogger)
	log(common.LogLevelInfo, "this endpoint logs a user out")
	w.Write([]byte("delete session"))
}
