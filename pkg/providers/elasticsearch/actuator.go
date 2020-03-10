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

package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/service"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/imagevector"
	"github.com/gardener/gardener-extensions/pkg/controller"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/operation/shoot"
	"github.com/gardener/gardener/pkg/utils/chart"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"strings"
	"time"
)

// Reconcile reconciles the auditlog extension for the elastic search provider
// Default
func (p *Provider) Reconcile(ctx context.Context, ex *extensionsv1alpha1.Extension) error {
	auditConfig := &service.Configuration{}
	if _, _, err := p.decoder.Decode(ex.Spec.ProviderConfig.Raw, nil, auditConfig); err != nil {
		return fmt.Errorf("failed to decode provider providerConfig: %+v", err)
	}
	providerConfig := &Configuration{}
	if err := yaml.Unmarshal(auditConfig.BackendProviderConfig, providerConfig); err != nil {
		return err
	}

	// default index to auditlog
	if providerConfig.Index == "" {
		providerConfig.Index = "auditlog"
	}

	if providerConfig.Endpoint != "" {
		return marshalExtension(ex, auditConfig, providerConfig)
	}

	if err := p.ensureAuditlogConfig(ctx, ex, auditConfig, providerConfig); err != nil {
		return err
	}
	return p.ensureGrafanaDashboard(ctx, ex, providerConfig)
}

// Delete removes the possibly deployed managed resource
func (p *Provider) Delete(ctx context.Context, ex *extensionsv1alpha1.Extension) error {
	secret := &corev1.Secret{}
	secret.Name = GrafanaSecretName
	secret.Namespace = ex.GetNamespace()
	if err := p.k8sClient.Delete(ctx, secret); err != nil {
		return err
	}

	if err := controller.DeleteManagedResource(ctx, p.k8sClient, ex.GetNamespace(), GrafanaDeploymentName); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return controller.WaitUntilManagedResourceDeleted(timeoutCtx, p.k8sClient, ex.GetNamespace(), GrafanaDeploymentName)
}

func (p *Provider) ensureAuditlogConfig(ctx context.Context, ex *extensionsv1alpha1.Extension, auditConfig *service.Configuration, providerConfig *Configuration) error {
	// try to use existing elasticsearch logging
	esService := &corev1.Service{}
	if err := p.k8sClient.Get(ctx, client.ObjectKey{Name: v1beta1constants.StatefulSetNameElasticSearch, Namespace: ex.GetNamespace()}, esService); err != nil {
		return errors.Wrapf(err, "unable to get service %s", v1beta1constants.StatefulSetNameElasticSearch)
	}
	// todo: use "db" port
	port, err := getPortByName("db", esService.Spec.Ports)
	if err != nil {
		return err
	}
	providerConfig.Endpoint = fmt.Sprintf("http://%s.%s:%d", esService.GetName(), esService.GetNamespace(), port)

	esSecret := &corev1.Secret{}
	if err := p.k8sClient.Get(ctx, client.ObjectKey{Name: "logging-ingress-credentials", Namespace: ex.GetNamespace()}, esSecret); err != nil {
		return errors.Wrapf(err, "unable to find elastic search secret")
	}
	providerConfig.Username = string(esSecret.Data["username"])
	providerConfig.Password = string(esSecret.Data["password"])

	return marshalExtension(ex, auditConfig, providerConfig)
}

func (p *Provider) ensureGrafanaDashboard(ctx context.Context, ex *extensionsv1alpha1.Extension, providerConfig *Configuration) error {
	p.log.Info("Ensuring Grafana dashboard", "namespace", ex.GetNamespace())

	cluster, err := controller.GetCluster(ctx, p.k8sClient, ex.GetNamespace())
	if err != nil {
		return err
	}

	// ensure grafana dashboard secret
	secret := &corev1.Secret{}
	secret.Name = GrafanaSecretName
	secret.Namespace = ex.GetNamespace()
	if err := p.k8sClient.Get(ctx, client.ObjectKey{Name: GrafanaSecretName, Namespace: ex.GetNamespace()}, secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		p.log.Info("Generating new grafana dashboard secret", "namespace", ex.GetNamespace())
		secret.StringData = map[string]string{
			"admin-user":     "admin",
			"admin-password": "test",
		}
		if err := p.k8sClient.Create(ctx, secret); err != nil {
			return err
		}
	}

	values := map[string]interface{}{
		"replicaCount": 1,
		"admin": map[string]string{
			"existingSecret": GrafanaSecretName,
		},
		"ingress": map[string]interface{}{
			"hosts": []map[string]interface{}{
				{
					"hostName":   p.ComputeIngressHost("ag", cluster),
					"secretName": fmt.Sprintf("%s-tls", GrafanaDeploymentName),
				},
			},
		},
		"datasources": map[string]interface{}{
			"datasources.yaml": map[string]interface{}{
				"apiVersion": 1,
				"datasources": []map[string]interface{}{{
					"name":              "Logging",
					"type":              "elasticsearch",
					"url":               providerConfig.Endpoint,
					"basicAuth":         true,
					"basicAuthUser":     providerConfig.Username,
					"basicAuthPassword": providerConfig.Password,
					"access":            "proxy",
					"isDefault":         true,
					"database":          providerConfig.Index,
					"jsonData": map[string]interface{}{
						"esVersion": 6,
						"timeField": "RequestReceivedTimestamp",
					},
				}},
			},
		},
	}
	if cluster.Shoot.Spec.Hibernation != nil && cluster.Shoot.Spec.Hibernation.Enabled != nil && *cluster.Shoot.Spec.Hibernation.Enabled {
		values["replicaCount"] = 0
	}

	values, err = chart.InjectImages(values, imagevector.ImageVector(), []string{GrafanaImageName})
	if err != nil {
		return fmt.Errorf("failed to find image version for %s: %v", GrafanaImageName, err)
	}
	return controller.CreateManagedResourceFromFileChart(
		ctx, p.k8sClient, ex.GetNamespace(), GrafanaDeploymentName, "seed",
		p.renderer, filepath.Join(ChartsPath, GrafanaChartName), GrafanaDeploymentName,
		values, nil,
	)
}

func getPortByName(name string, ports []corev1.ServicePort) (int32, error) {
	for _, port := range ports {
		if port.Name == name {
			return port.Port, nil
		}
	}
	return 0, errors.Errorf("unable to find port for %s", name)
}

// ComputeIngressHost computes the host for a given prefix.
func (p *Provider) ComputeIngressHost(prefix string, cluster *controller.Cluster) string {
	shortID := strings.Replace(cluster.Shoot.Status.TechnicalID, shoot.TechnicalIDPrefix, "", 1)
	return fmt.Sprintf("%s-%s.%s", prefix, shortID, cluster.Seed.Spec.DNS.IngressDomain)
}

func marshalExtension(ex *extensionsv1alpha1.Extension, auditConfig *service.Configuration, providerConfig *Configuration) error {
	providerData, err := json.Marshal(providerConfig)
	if err != nil {
		return errors.Wrap(err, "unable to marshal auditlog provider config")
	}
	auditConfig.BackendProviderConfig = json.RawMessage(providerData)
	auditlogData, err := json.Marshal(auditConfig)
	if err != nil {
		return errors.Wrap(err, "unable to marshal auditlog config")
	}
	ex.Spec.ProviderConfig.Raw = auditlogData
	return nil
}
