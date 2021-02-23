// Defines an HTTP server that reports the readiness of the ocs-osd-operator,
// and helper functions to Set/Unset its readiness state.
// For this to work, ensure the following is added to the Deployment definition
// in the CSV:
//                readinessProbe:
//                  httpGet:
//                    path: /readyz
//                    port: 8081
//                  initialDelaySeconds: 5
//                  periodSeconds: 10

package utils

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

const (
	PortStr = ":8081"
)

type ReadinessServer struct {
	log           logr.Logger
	operatorReady bool
	lastUpdated   time.Time
	sync.Mutex
}

func NewReadinessServer(log logr.Logger) *ReadinessServer {
	return &ReadinessServer{
		log: log,
	}
}

func (r *ReadinessServer) StartReadinessServer() {
	http.HandleFunc("/readyz", func(w http.ResponseWriter, req *http.Request) {
		r.log.Info("readiness is being probed")
		r.Lock()
		operatorReady := r.operatorReady
		lastUpdated := r.lastUpdated
		r.Unlock()
		// Readiness logic is defined here.
		// From k8s documentation:
		// "Any code greater than or equal to 200 and less than 400 indicates success."
		// [indicates that the deployment is ready]
		// "Any other code indicates failure."
		// [indicates that the deployment is not ready]
		if operatorReady {
			w.WriteHeader(200)
			r.log.Info("operator is ready")
		} else {
			w.WriteHeader(418)
			r.log.Info("operator is not ready")
		}
		// For testing purposes, writting a timestamp in the body of the response
		// This will be used to track when the readiness was last updated.
		w.Write([]byte(lastUpdated.Format(time.StampMicro)))
	})

	r.log.Info("starting readiness server")
	err := http.ListenAndServe(PortStr, nil)
	if err != nil {
		r.log.Error(err, "error occurred in readiness server")
	}
}

func (r *ReadinessServer) SetReady() {
	r.Lock()
	r.operatorReady = true
	r.lastUpdated = time.Now()
	r.Unlock()
}

func (r *ReadinessServer) UnsetReady(reason string) {
	r.Lock()
	r.operatorReady = false
	r.lastUpdated = time.Now()
	r.Unlock()
	r.log.Info(reason)
}
