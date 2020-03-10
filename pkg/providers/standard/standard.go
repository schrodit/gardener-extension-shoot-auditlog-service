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

package standard

import (
	"context"
	"fmt"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/provider"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apiserver/pkg/apis/audit"
)

type Provider struct {
	log logr.Logger
}

var _ provider.Interface = &Provider{}

func (p *Provider) New() (provider.Interface, error) {
	return &Provider{}, nil
}

func (p *Provider) Name() string {
	return "standard"
}

func (p *Provider) InjectLogger(log logr.Logger) error {
	p.log = log
	return nil
}

func (p *Provider) Log(events *audit.EventList) error {
	for _, event := range events.Items {
		obj := fmt.Sprintf("%s/%s - %s/%s", event.ObjectRef.APIGroup, event.ObjectRef.APIVersion, event.ObjectRef.Name, event.ObjectRef.Namespace)
		p.log.WithName("audit").Info(obj, "level", event.Level, "user", event.User.Username)
	}
	return nil
}

// Reconcile noop reconcile function
func (p *Provider) Reconcile(ctx context.Context, ex *extensionsv1alpha1.Extension) error { return nil }

// Delete noop delete function
func (p *Provider) Delete(ctx context.Context, ex *extensionsv1alpha1.Extension) error { return nil }
