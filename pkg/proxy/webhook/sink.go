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
	apisconfig "github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/proxy"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/provider"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/providers"
	"github.com/go-logr/logr"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/apiserver/pkg/apis/audit/install"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

type sink struct {
	log      logr.Logger
	decoder  runtime.Decoder
	provider provider.Interface
}

// NewSink creates a new Sink objects that can handle kubernetes auditlog events
func NewSink(log logr.Logger, config *apisconfig.Configuration) (http.Handler, error) {
	auditScheme := runtime.NewScheme()
	install.Install(auditScheme)

	p, err := providers.ProviderFactory.Get(config.Provider)
	if err != nil {
		return nil, err
	}
	if _, err = provider.BackendConfigInto(config.ProviderConfig, p); err != nil {
		return nil, err
	}
	if _, err = inject.LoggerInto(log, p); err != nil {
		return nil, err
	}

	log.Info("Provider successfully loaded", "provider", config.Provider)

	return &sink{
		log:      log,
		decoder:  serializer.NewCodecFactory(auditScheme).UniversalDecoder(),
		provider: p,
	}, nil
}

// HandleAudit is the handler
func (s *sink) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	raw, err := ioutil.ReadAll(req.Body)
	if err != nil {
		s.log.Error(err, "unable to read body of request")
		http.Error(w, "unable to read content", http.StatusBadRequest)
		return
	}

	eventList := &audit.EventList{}
	if _, _, err := s.decoder.Decode(raw, nil, eventList); err != nil {
		s.log.Error(err, "unable to decode eventList")
		http.Error(w, "unable to decode eventList", http.StatusBadRequest)
		return
	}

	s.log.V(8).Info("Parsed event list", "events", eventList)

	if err := s.provider.Log(eventList); err != nil {
		s.log.Error(err, "unable to log eventList")
		http.Error(w, "unable log eventList", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}
