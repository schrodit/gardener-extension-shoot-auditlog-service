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

package cmd

import (
	"errors"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/shootauditlog"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/webhook/controlplane"
	"io/ioutil"

	apisconfig "github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/config/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/config/validation"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/controller"
	controllerconfig "github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/controller/config"
	healthcheckcontroller "github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/controller/healthcheck"

	"github.com/gardener/gardener-extensions/pkg/controller/cmd"
	extensionshealthcheckcontroller "github.com/gardener/gardener-extensions/pkg/controller/healthcheck"
	healthcheckconfig "github.com/gardener/gardener-extensions/pkg/controller/healthcheck/config"
	webhookcmd "github.com/gardener/gardener-extensions/pkg/webhook/cmd"
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
type AuditlogServiceOptions struct {
	ConfigLocation string
	config         *AuditlogServiceConfig
}

// AddFlags implements Flagger.AddFlags.
func (o *AuditlogServiceOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ConfigLocation, "config", "", "Path to cert service configuration")
}

// Complete implements Completer.Complete.
func (o *AuditlogServiceOptions) Complete() error {
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

	o.config = &AuditlogServiceConfig{
		config: config,
	}

	return nil
}

// Completed returns the decoded CertificatesServiceConfiguration instance. Only call this if `Complete` was successful.
func (o *AuditlogServiceOptions) Completed() *AuditlogServiceConfig {
	return o.config
}

// AuditlogProxyConfig contains configuration information about the certificate service.
type AuditlogServiceConfig struct {
	config apisconfig.Configuration
}

// Apply applies the AuditlogProxyOptions to the passed ControllerOptions instance.
func (c *AuditlogServiceConfig) Apply(config *controllerconfig.Config) {
	config.Configuration = c.config
}

// ControllerSwitches are the cmd.SwitchOptions for the provider controllers.
func ControllerSwitches() *cmd.SwitchOptions {
	return cmd.NewSwitchOptions(
		cmd.Switch(shootauditlog.ControllerName, controller.AddToManager),
		cmd.Switch(extensionshealthcheckcontroller.ControllerName, healthcheckcontroller.AddToManager),
	)
}

// WebhookSwitchOptions are the webhookcmd.SwitchOptions for the provider webhooks.
func WebhookSwitchOptions() *webhookcmd.SwitchOptions {
	return webhookcmd.NewSwitchOptions(
		webhookcmd.Switch(shootauditlog.ControllerName, controlplane.AddToManager),
	)
}

func (c *AuditlogServiceConfig) ApplyHealthCheckConfig(config *healthcheckconfig.HealthCheckConfig) {
	if c.config.HealthCheckConfig != nil {
		*config = *c.config.HealthCheckConfig
	}
}
