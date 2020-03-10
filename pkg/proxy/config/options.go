// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package config

import (
	"errors"
	"io/ioutil"

	apisconfig "github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/proxy"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/proxy/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/proxy/validation"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var (
	scheme  *runtime.Scheme
	decoder runtime.Decoder
)

func init() {
	scheme = runtime.NewScheme()
	utilruntime.Must(apisconfig.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
}

// AuditlogProxyOptions holds options related to the certificate service.
type AuditlogProxyOptions struct {
	ConfigLocation string
	config         *apisconfig.Configuration
}

// AddFlags implements Flagger.AddFlags.
func (o *AuditlogProxyOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ConfigLocation, "config", "", "Path to auditlog proxy configuration")
}

// Complete implements Completer.Complete.
func (o *AuditlogProxyOptions) Complete() error {
	if o.ConfigLocation == "" {
		return errors.New("config location is not set")
	}
	data, err := ioutil.ReadFile(o.ConfigLocation)
	if err != nil {
		return err
	}

	config := apisconfig.Configuration{}
	_, _, err = decoder.Decode(data, nil, &config)
	if err != nil {
		return err
	}

	if errs := validation.ValidateConfiguration(&config); len(errs) > 0 {
		return errs.ToAggregate()
	}

	o.config = &config

	return nil
}

// Completed returns the decoded CertificatesServiceConfiguration instance. Only call this if `Complete` was successful.
func (o *AuditlogProxyOptions) Completed() *apisconfig.Configuration {
	return o.config
}
