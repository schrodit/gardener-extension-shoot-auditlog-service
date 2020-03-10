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

package provider

import (
	"github.com/gardener/gardener-extensions/pkg/controller/extension"
	"k8s.io/apiserver/pkg/apis/audit"
)

// Interface is the abstraction for multiple backend providers
type Interface interface {
	extension.Actuator

	Name() string
	New() (Interface, error)
	Log(events *audit.EventList) error
}

// BackendConfig is used by the auditlog extension to inject backend Config
type BackendConfig interface {
	InjectBackendConfig([]byte) error
}

// BackendConfigInto will set config on i and return the result if it implements Config.  Returns
// false if i does not implement Config.
func BackendConfigInto(config []byte, i interface{}) (bool, error) {
	if s, ok := i.(BackendConfig); ok {
		return true, s.InjectBackendConfig(config)
	}
	return false, nil
}
