// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package webhook

import (
	"context"
	"fmt"
	apisconfig "github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/proxy"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

func Run(log logr.Logger, config *apisconfig.Configuration) error {
	ctx := context.Background()
	defer ctx.Done()

	sinkHandler, err := NewSink(log.WithName("sink"), config)
	if err != nil {
		return err
	}

	router := mux.NewRouter()
	router.Use(getTraceMiddleware(log))
	router.PathPrefix("/").Handler(sinkHandler).Methods(http.MethodPost)
	router.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }).Methods(http.MethodGet)
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })

	serverHTTP := &http.Server{Addr: fmt.Sprintf(":%d", config.WebhookConfiguration.HTTPPort), Handler: router}
	serverHTTPS := &http.Server{Addr: fmt.Sprintf(":%d", config.WebhookConfiguration.HTTPSPort), Handler: router}

	go func() {
		log.Info("starting webhook http server", "port", serverHTTP.Addr)
		if err := serverHTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(err, "unable to start HTTP webhook server")
		}
	}()

	if config.WebhookConfiguration.HTTPSPort != 0 {
		go func() {
			log.Info("starting webhook https server", "port", serverHTTPS.Addr)
			if err := serverHTTPS.ListenAndServeTLS(config.WebhookConfiguration.TLS.CertFile, config.WebhookConfiguration.TLS.KeyFile); err != nil && err != http.ErrServerClosed {
				log.Error(err, "unable to start webhook https server")
			}
		}()
	}

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := serverHTTP.Shutdown(ctx); err != nil {
		log.Error(err, "unable to shut down HTTP server")
	}
	if err := serverHTTPS.Shutdown(ctx); err != nil {
		log.Error(err, "unable to shut down HTTPS server")
	}
	log.Info("HTTP(S) servers stopped.")
	return nil
}

func getTraceMiddleware(log logr.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.WithName("trace").V(5).Info(r.RequestURI, "method", r.Method)
			next.ServeHTTP(w, r)
		})
	}
}
