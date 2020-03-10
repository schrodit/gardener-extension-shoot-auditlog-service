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

package v1alpha1

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Configuration contains information about the auditlog service configuration.
type Configuration struct {
	metav1.TypeMeta `json:",inline"`

	// BackendProvider specifies the provider for the audit log proxy where the logs are persisted
	BackendProvider string `json:"backendProvider"`

	// BackendProviderConfig is the backend provider specific configuration
	BackendProviderConfig json.RawMessage `json:"backendProviderConfig"`

	// Policy is the raw audit log policy.
	// Be aware that k8s clusters <=1.11 do not support "audit.k8s.io/v1"
	Policy runtime.RawExtension `json:"policy"`
}
