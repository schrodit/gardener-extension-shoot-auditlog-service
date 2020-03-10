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

package app

import (
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/shootauditlog"
	webhookcmd "github.com/gardener/gardener-extensions/pkg/webhook/cmd"
	"os"

	auditlogservicecmd "github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/cmd"

	controllercmd "github.com/gardener/gardener-extensions/pkg/controller/cmd"
)

// ExtensionName is the name of the extension.
const ExtensionName = "extension-auditlog-service"

// Options holds configuration passed to the Certificate Service controller.
type Options struct {
	auditlogOptions    *auditlogservicecmd.AuditlogServiceOptions
	restOptions        *controllercmd.RESTOptions
	managerOptions     *controllercmd.ManagerOptions
	controllerOptions  *controllercmd.ControllerOptions
	webhookOptions     *webhookcmd.AddToManagerOptions
	healthOptions      *controllercmd.ControllerOptions
	controllerSwitches *controllercmd.SwitchOptions
	reconcileOptions   *controllercmd.ReconcilerOptions
	optionAggregator   controllercmd.OptionAggregator
}

// NewOptions creates a new Options instance.
func NewOptions() *Options {
	options := &Options{
		auditlogOptions: &auditlogservicecmd.AuditlogServiceOptions{},
		restOptions:     &controllercmd.RESTOptions{},
		managerOptions: &controllercmd.ManagerOptions{
			// These are default values.
			LeaderElection:          true,
			LeaderElectionID:        controllercmd.LeaderElectionNameID(ExtensionName),
			LeaderElectionNamespace: os.Getenv("LEADER_ELECTION_NAMESPACE"),
			WebhookServerPort:       443,
			WebhookCertDir:          "/tmp/gardener-extensions-cert",
		},
		controllerOptions: &controllercmd.ControllerOptions{
			// This is a default value.
			MaxConcurrentReconciles: 5,
		},
		healthOptions: &controllercmd.ControllerOptions{
			// This is a default value.
			MaxConcurrentReconciles: 5,
		},
		controllerSwitches: auditlogservicecmd.ControllerSwitches(),
		reconcileOptions:   &controllercmd.ReconcilerOptions{},
	}

	// options for the webhook server
	webhookServerOptions := &webhookcmd.ServerOptions{
		Namespace: os.Getenv("WEBHOOK_CONFIG_NAMESPACE"),
	}
	webhookSwitches := auditlogservicecmd.WebhookSwitchOptions()
	options.webhookOptions = webhookcmd.NewAddToManagerOptions(shootauditlog.Type, webhookServerOptions, webhookSwitches)

	options.optionAggregator = controllercmd.NewOptionAggregator(
		options.auditlogOptions,
		options.restOptions,
		options.managerOptions,
		options.controllerOptions,
		controllercmd.PrefixOption("healthcheck-", options.healthOptions),
		options.controllerSwitches,
		options.reconcileOptions,
		options.webhookOptions,
	)

	return options
}
