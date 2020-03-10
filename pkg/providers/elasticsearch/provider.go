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

package elasticsearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/provider"
	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/client-go/rest"
	"net/http"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type Provider struct {
	log       logr.Logger
	client    *http.Client
	renderer  chartrenderer.Interface
	k8sClient client.Client
	config    *Configuration
	decoder   runtime.Decoder
}

// Configuration is the elasticsearch provider specific configuration
type Configuration struct {
	Endpoint string `json:"endpoint"`
	Username string `json:"username"`
	Password string `json:"password"`
	Index    string `json:"index"`
}

// GrafanaImageName is the name of the grafana image
const GrafanaImageName = "grafana"

// ChartsPath is the path to the charts
var ChartsPath = filepath.Join("charts", "internal", "elasticsearch")

// GrafanaChartName is the name of the chart for the grafana
const GrafanaChartName = "shoot-auditlog-grafana"

// GrafanaDeploymentName is the name of the audilog grafana deployment
const GrafanaDeploymentName = "shoot-auditlog-grafana"

// GrafanaSecretName is the name of the secret that contains the admin credentials for the grafana dashboard
const GrafanaSecretName = "shoot-auditlog-grafana-admin-secret"

var _ provider.Interface = &Provider{}

func (p *Provider) New() (provider.Interface, error) {
	return &Provider{
		client: http.DefaultClient,
	}, nil
}

func (p *Provider) Name() string {
	return "elasticsearch"
}

func (p *Provider) InjectClient(k8sClient client.Client) error {
	p.k8sClient = k8sClient
	return nil
}

func (p *Provider) InjectConfig(restConfig *rest.Config) error {
	renderer, err := chartrenderer.NewForConfig(restConfig)
	if err != nil {
		return errors.Wrap(err, "could not create chart renderer")
	}
	p.renderer = renderer
	return nil
}

func (p *Provider) InjectBackendConfig(rawConfig []byte) error {
	config := &Configuration{}
	if err := yaml.Unmarshal(rawConfig, config); err != nil {
		return err
	}
	p.config = config
	return nil
}

func (p *Provider) InjectLogger(log logr.Logger) error {
	p.log = log
	return nil
}

// InjectScheme injects the given scheme into the reconciler.
func (p *Provider) InjectScheme(scheme *runtime.Scheme) error {
	p.decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
	return nil
}

func (p *Provider) Log(events *audit.EventList) error {
	if p.config == nil {
		return errors.New("configuration is not defined")
	}
	bulk := bytes.NewBuffer([]byte{})

	for _, event := range events.Items {

		// omit request and response object due to elasticsearch indexing issues
		// Could not dynamically add mapping for field [app.kubernetes.io/instance]. Existing mapping for [ResponseObject.items.metadata.labels.app] must be of type object but found [text].
		event.RequestObject = nil
		event.ResponseObject = nil

		obj, err := json.Marshal(event)
		if err != nil {
			return err
		}
		bulk.WriteString(fmt.Sprintf(`{ "index": { "_index": "%s", "_type": "_doc" } }`, p.config.Index))
		bulk.WriteRune('\n')
		bulk.Write(obj)
		bulk.WriteRune('\n')
	}

	if err := p.bulk(bulk.Bytes()); err != nil {
		return err
	}
	p.log.Info("Successfully ingested logs", "events", len(events.Items), "index", p.config.Index)
	return nil
}
