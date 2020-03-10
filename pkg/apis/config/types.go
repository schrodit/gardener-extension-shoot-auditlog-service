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
	healthcheckconfig "github.com/gardener/gardener-extensions/pkg/controller/healthcheck/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
)

// AuditlogProxyImageName is the name of the Auditlog Proxy image in the image vector.
const AuditlogProxyImageName = "auditlog-proxy"

// ChartsPath is the path to the charts
var ChartsPath = filepath.Join("charts", "internal")

// AuditlogProxyChartName is the name of the chart for the auditlog proxy
const AuditlogProxyChartName = "shoot-auditlog-proxy"

// AuditlogProxyResourceName is the name of the chart for the auditlog proxy
const AuditlogProxyResourceName = "shoot-auditlog-proxy"

// AuditlogProxyServiceName is the name of the svc where the shoot auditlog proxy can be reached
const AuditlogProxyServiceName = "shoot-auditlog-proxy"

// AuditlogKubecfgSecretName is the name of the secret for the auditlog webhook kubeconfig
const AuditlogKubecfgSecretName = "extension-shoot-auditlog-kubecfg"

// AuditlogKubecfgSecretName is the name of the secret for the auditlog webhook kubeconfig
const AuditlogPolicyConfigMapName = "extension-shoot-auditlog-policy"

// AuditlogExtensionLabel is the label for the shoot namespace that inidcates if the extension is activated
const AuditlogExtensionLabel = "extension.gardener.cloud/shoot-auditlog"

// AllowAuditlogProxyNetworkPolicyLabel is the label to allow traffic to the auditlog proxy
const AllowAuditlogProxyNetworkPolicyLabel = "networking.gardener.cloud/to-auditlog-proxy"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Configuration contains information about the auditlog service configuration.
type Configuration struct {
	metav1.TypeMeta `json:",inline"`

	// HealthCheckConfig is the config for the health check controller
	// +optional
	HealthCheckConfig *healthcheckconfig.HealthCheckConfig `json:"healthCheckConfig,omitempty"`
}
