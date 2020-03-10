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

package proxy

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Configuration contains information about the auditlog service configuration.
type Configuration struct {
	metav1.TypeMeta `json:",inline"`

	// Provider is the storage provider to store the auditlogs
	Provider string `json:"provider"`

	// ProviderConfig is the provider specific storage configuration
	// +optional
	ProviderConfig json.RawMessage `json:"providerConfig"`

	// WebhookConfiguration holds the webhook specific configuration
	WebhookConfiguration WebhookConfiguration `json:"webhookConfiguration"`
}

// WebhookConfiguration contains information about the proxy webhook endpoint
type WebhookConfiguration struct {
	HTTPPort  int `json:"httpPort"`
	HTTPSPort int `json:"httpsPort"`

	TLS TLSConfiguration `json:"tls"`
}

// TLSConfiguration contains the cert and key for tls
type TLSConfiguration struct {
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
}
