package reporter

import (
	"fmt"
	"opsicle/pkg/approver"
	"time"

	"github.com/sirupsen/logrus"
)

func startApproverReporter(appName string, opts *healthcheckOpts) error {
	logrus.Debugf("connecting to approver service...")
	approverClient, err := approver.NewClient(approver.NewClientOpts{
		ApproverUrl: "http://localhost:12345",
	})
	if err != nil {
		return fmt.Errorf("failed to create client for approver service: %w", err)
	}
	if err := approverClient.Ping(); err != nil {
		logrus.Warnf("approver service is down: %s", err)
		opts.status.Set(0)
	} else {
		logrus.Infof("connected to approver service")
		opts.status.Set(1)
	}
	go func() {
		for {
			select {
			case <-opts.stopper:
				return
			default:
				<-time.After(opts.interval)
				if err := approverClient.Ping(); err != nil {
					opts.status.Set(0)
					logrus.Warnf("approver service is down: %s", err)
					continue
				}
				opts.status.Set(1)
			}
		}
	}()
	return nil
}
