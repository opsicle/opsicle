package approver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type StartHttpServerOpts struct {
	Addr string
}

func StartHttpServer(opts StartHttpServerOpts) error {
	r := mux.NewRouter()
	r.HandleFunc("/approve", handleApprovalRequest).Methods("POST")
	logrus.Infof("Starting HTTP server on %s", opts.Addr)

	if err := http.ListenAndServe(opts.Addr, r); err != nil {
		return fmt.Errorf("failed to start server: %s", err)
	}
	return nil
}

func handleApprovalRequest(w http.ResponseWriter, r *http.Request) {
	var req ApprovalRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	err = RedisCache.Client.Set(req.Id, string(body), 0).Err()
	if err != nil {
		http.Error(w, "failed to store request", http.StatusInternalServerError)
		return
	}

	err = TelegramApprover.SendApproval(req, SendApprovalOpts{
		Chat: req.Chat,
	})
	if err != nil {
		http.Error(w, "failed to send Telegram message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("approval request submitted"))
}
