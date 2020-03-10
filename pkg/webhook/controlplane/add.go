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

package controlplane

import (
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/shootauditlog"
	extensionswebhook "github.com/gardener/gardener-extensions/pkg/webhook"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane/genericmutator"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var logger = log.Log.WithName("shoot-auditlog-controlplane-webhook")

// AddToManager creates a webhook and adds it to the manager.
func AddToManager(mgr manager.Manager) (*extensionswebhook.Webhook, error) {
	logger.Info("Adding webhook to manager")

	fciCodec := controlplane.NewFileContentInlineCodec()
	handler, err := extensionswebhook.NewHandler(mgr,
		[]runtime.Object{&appsv1.Deployment{}},
		genericmutator.NewMutator(NewEnsurer(logger), controlplane.NewUnitSerializer(),
			controlplane.NewKubeletConfigCodec(fciCodec), fciCodec, logger),
		logger,
	)
	if err != nil {
		return nil, err
	}

	return &extensionswebhook.Webhook{
		Name:    shootauditlog.Type,
		Types:   []runtime.Object{&appsv1.Deployment{}},
		Target:  extensionswebhook.TargetSeed,
		Path:    shootauditlog.Type,
		Webhook: &admission.Webhook{Handler: handler},
		Selector: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{Key: config.AuditlogExtensionLabel, Operator: metav1.LabelSelectorOpExists},
			},
		},
	}, nil
}
