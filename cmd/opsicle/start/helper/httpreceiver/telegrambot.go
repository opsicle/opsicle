package httpreceiver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"opsicle/internal/cli"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "addr",
		DefaultValue: "0.0.0.0:54321",
		Usage:        "defines the interface address which the service should listen on",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "httpreceiver",
	Aliases: []string{"http"},
	Short:   "Runs an echo service",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		http.HandleFunc("/", logRequestHandler)

		addr := viper.GetString("addr")
		logrus.Infof("starting server on %s", addr)

		if err := http.ListenAndServe(addr, nil); err != nil {
			logrus.Errorf("failed to start server: %v", err)
		}
		return nil
	},
}

func logRequestHandler(w http.ResponseWriter, r *http.Request) {
	var bodyBuf bytes.Buffer
	bodyContent := "(unreadable)"
	if _, err := io.Copy(&bodyBuf, r.Body); err != nil {
		logrus.WithError(err).Warn("Failed to read request body")
	} else {
		bodyContent = bodyBuf.String()
	}
	defer r.Body.Close()

	requestDetails := map[string]interface{}{
		"method":     r.Method,
		"url":        r.URL.String(),
		"proto":      r.Proto,
		"host":       r.Host,
		"remoteAddr": r.RemoteAddr,
		"headers":    r.Header,
		"body":       bodyContent,
	}

	data, err := json.MarshalIndent(requestDetails, "", "  ")
	if err != nil {
		logrus.Errorf("failed to marshal request to json: %s", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	logrus.Infof("incoming request:\n%s", string(data))

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}
