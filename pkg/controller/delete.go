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

package controller

import (
	"context"
	"fmt"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/service"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/providers"
	"github.com/gardener/gardener-extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"time"
)

// Delete the Extension resource.
func (a *actuator) Delete(ctx context.Context, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()
	a.logger.Info("Component is being deleted", "component", "auditlog", "namespace", namespace)

	return a.deleteSeedResources(ctx, ex)
}

func (a *actuator) deleteSeedResources(ctx context.Context, ex *extensionsv1alpha1.Extension) error {
	a.logger.Info("Deleting managed resource for seed", "namespace", ex.GetNamespace())

	secret := &corev1.Secret{}
	secret.SetName(config.AuditlogKubecfgSecretName)
	secret.SetNamespace(ex.GetNamespace())
	if err := a.client.Delete(ctx, secret); client.IgnoreNotFound(err) != nil {
		return err
	}

	cm := &corev1.ConfigMap{}
	cm.SetName(config.AuditlogPolicyConfigMapName)
	cm.SetNamespace(ex.GetNamespace())
	if err := a.client.Delete(ctx, cm); client.IgnoreNotFound(err) != nil {
		return err
	}

	if err := a.removeNamespaceLabel(ctx, ex.GetNamespace()); err != nil {
		return err
	}

	if err := controller.DeleteManagedResource(ctx, a.client, ex.GetNamespace(), config.AuditlogProxyResourceName); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := controller.WaitUntilManagedResourceDeleted(timeoutCtx, a.client, ex.GetNamespace(), config.AuditlogProxyResourceName); err != nil {
		return err
	}
	return a.deleteBackendProvider(ctx, ex)
}

func (a *actuator) deleteBackendProvider(ctx context.Context, ex *extensionsv1alpha1.Extension) error {
	auditConfig := &service.Configuration{}
	if _, _, err := a.decoder.Decode(ex.Spec.ProviderConfig.Raw, nil, auditConfig); err != nil {
		return fmt.Errorf("failed to decode provider config: %+v", err)
	}

	p, err := providers.ProviderFactory.Get(auditConfig.BackendProvider)
	if err != nil {
		return nil
	}
	if _, err := inject.ClientInto(a.client, p); err != nil {
		return err
	}
	if _, err := inject.LoggerInto(a.logger, p); err != nil {
		return err
	}
	return p.Delete(ctx, ex)
}

func (a *actuator) removeNamespaceLabel(ctx context.Context, namespace string) error {
	a.logger.Info("Removing auditlog label from namespace", "namespace", namespace)
	ns := &corev1.Namespace{}
	if err := a.client.Get(ctx, client.ObjectKey{Name: namespace}, ns); err != nil {
		return err
	}
	if ns.Labels == nil {
		return nil
	}

	if _, ok := ns.Labels[config.AuditlogExtensionLabel]; !ok {
		return nil
	}

	delete(ns.Labels, config.AuditlogExtensionLabel)
	return a.client.Update(ctx, ns)
}
