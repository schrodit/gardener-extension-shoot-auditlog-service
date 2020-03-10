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

package controller

import (
	"context"
	"fmt"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/service"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/service/validation"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/imagevector"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/providers"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/webhook/controlplane"
	"github.com/gardener/gardener-extensions/pkg/controller"
	"github.com/gardener/gardener-extensions/pkg/controller/extension"
	extensionutil "github.com/gardener/gardener-extensions/pkg/util"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane/genericmutator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/utils/chart"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/yaml"
	"strings"
	"time"
)

// ActuatorName is the name of the Certificate Service actuator.
const ActuatorName = "shoot-auditlog-actuator"

// NewActuator returns an actuator responsible for Extension resources.
func NewActuator(config config.Configuration) extension.Actuator {
	return &actuator{
		logger:              log.Log.WithName(ActuatorName),
		serviceConfig:       config,
		controlplaneEnsurer: controlplane.NewEnsurer(log.Log),
	}
}

type actuator struct {
	client              client.Client
	config              *rest.Config
	scheme              *runtime.Scheme
	decoder             runtime.Decoder
	controlplaneEnsurer genericmutator.Ensurer

	serviceConfig config.Configuration

	logger logr.Logger
}

// Reconcile the Extension resource.
func (a *actuator) Reconcile(ctx context.Context, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	cluster, err := controller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return err
	}

	return a.createSeedResources(ctx, ex, cluster, namespace)
}

// InjectConfig injects the rest config to this actuator.
func (a *actuator) InjectConfig(config *rest.Config) error {
	a.config = config
	return nil
}

// InjectClient injects the controller runtime client into the reconciler.
func (a *actuator) InjectClient(client client.Client) error {
	a.client = client
	return a.controlplaneEnsurer.(inject.Client).InjectClient(client)
}

// InjectScheme injects the given scheme into the reconciler.
func (a *actuator) InjectScheme(scheme *runtime.Scheme) error {
	a.scheme = scheme
	a.decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
	return nil
}

func (a *actuator) createSeedResources(ctx context.Context, ex *extensionsv1alpha1.Extension, cluster *controller.Cluster, namespace string) error {
	if ex.Spec.ProviderConfig == nil {
		return nil
	}

	var (
		auditConfig = &service.Configuration{}
		checksums   = map[string]string{}
	)
	if _, _, err := a.decoder.Decode(ex.Spec.ProviderConfig.Raw, nil, auditConfig); err != nil {
		return fmt.Errorf("failed to decode provider config: %+v", err)
	}
	if errs := validation.ValidateConfiguration(auditConfig); len(errs) > 0 {
		return errs.ToAggregate()
	}

	if err := a.ensureNamespaceLabel(ctx, ex.Namespace); err != nil {
		return err
	}

	if err := a.ensureAuditPolicyConfig(ctx, ex.Namespace, auditConfig, checksums); err != nil {
		return err
	}

	if err := a.ensureWebhookKubeconfig(ctx, ex.GetNamespace(), checksums); err != nil {
		return err
	}

	if err := a.ensureBackendProvider(ctx, auditConfig, ex); err != nil {
		return err
	}

	if _, _, err := a.decoder.Decode(ex.Spec.ProviderConfig.Raw, nil, auditConfig); err != nil {
		return fmt.Errorf("failed to decode provider config: %+v", err)
	}
	if errs := validation.ValidateConfiguration(auditConfig); len(errs) > 0 {
		return errs.ToAggregate()
	}

	auditlogProxyValues := map[string]interface{}{
		"replicaCount": 1,
		"configuration": map[string]interface{}{
			"serverPortHttps": 443,
			"provider":        auditConfig.BackendProvider,
			"providerConfig":  auditConfig.BackendProviderConfig,
		},
		"svc": map[string]interface{}{
			"name": config.AuditlogProxyServiceName,
		},
		"tls": map[string]interface{}{
			"secretName": config.AuditlogKubecfgSecretName,
		},
		"podAnnotations": checksums,
		"additionalConfiguration": []string{
			"-v=5",
		},
	}

	if cluster.Shoot.Spec.Hibernation != nil && cluster.Shoot.Spec.Hibernation.Enabled != nil && *cluster.Shoot.Spec.Hibernation.Enabled {
		auditlogProxyValues["replicaCount"] = 0
	}

	auditlogProxyConfig, err := chart.InjectImages(auditlogProxyValues, imagevector.ImageVector(), []string{config.AuditlogProxyImageName})
	if err != nil {
		return fmt.Errorf("failed to find image version for %s: %v", config.AuditlogProxyImageName, err)
	}

	renderer, err := chartrenderer.NewForConfig(a.config)
	if err != nil {
		return errors.Wrap(err, "could not create chart renderer")
	}

	a.logger.Info("Deploy auditlog proxy", "component", "shoot-auditlog-proxy", "namespace", namespace)
	if err := a.createManagedResource(ctx, namespace, config.AuditlogProxyResourceName, renderer, config.AuditlogProxyChartName, auditlogProxyConfig, nil); err != nil {
		return err
	}

	return a.ensureKubeAPIServerDeployment(ctx, ex.GetNamespace())
}

func (a *actuator) ensureBackendProvider(ctx context.Context, auditConfig *service.Configuration, ex *extensionsv1alpha1.Extension) error {
	a.logger.Info("Ensuring backend provider", "namespace", ex.GetNamespace(), "provider", auditConfig.BackendProvider)

	p, err := providers.ProviderFactory.Get(auditConfig.BackendProvider)
	if err != nil {
		return nil
	}
	if _, err := inject.ClientInto(a.client, p); err != nil {
		return err
	}
	if _, err := inject.ConfigInto(a.config, p); err != nil {
		return err
	}
	if _, err := inject.SchemeInto(a.scheme, p); err != nil {
		return err
	}
	if _, err := inject.LoggerInto(a.logger, p); err != nil {
		return err
	}

	return p.Reconcile(ctx, ex)
}

func (a *actuator) createManagedResource(ctx context.Context, namespace, name string, renderer chartrenderer.Interface, chartName string, chartValues map[string]interface{}, injectedLabels map[string]string) error {
	return controller.CreateManagedResourceFromFileChart(
		ctx, a.client, namespace, name, "seed",
		renderer, filepath.Join(config.ChartsPath, chartName), chartName,
		chartValues, injectedLabels,
	)
}

func (a *actuator) ensureNamespaceLabel(ctx context.Context, namespace string) error {
	a.logger.Info("Ensuring auditlog label from namespace", "namespace", namespace)
	ns := &corev1.Namespace{}
	if err := a.client.Get(ctx, client.ObjectKey{Name: namespace}, ns); err != nil {
		return err
	}
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}

	if _, ok := ns.Labels[config.AuditlogExtensionLabel]; ok {
		return nil
	}

	ns.Labels[config.AuditlogExtensionLabel] = ""

	return a.client.Update(ctx, ns)
}

func (a *actuator) ensureAuditPolicyConfig(ctx context.Context, namespace string, auditConfig *service.Configuration, checksums map[string]string) error {
	a.logger.Info("Ensuring auditlog policy config", "namespace", namespace)
	cm := &corev1.ConfigMap{}
	cm.SetName(config.AuditlogPolicyConfigMapName)
	cm.SetNamespace(namespace)
	err := a.client.Get(ctx, client.ObjectKey{Name: config.AuditlogPolicyConfigMapName, Namespace: namespace}, cm)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	rawPolicyConfig, err := yaml.Marshal(auditConfig.Policy)
	if err != nil {
		return err
	}

	cm.Data = map[string]string{
		"audit-policy.yaml": string(rawPolicyConfig),
	}

	checksums[fmt.Sprintf("checksum/configmap-%s", config.AuditlogPolicyConfigMapName)] = extensionutil.ComputeChecksum(rawPolicyConfig)

	_, err = controllerutil.CreateOrUpdate(ctx, a.client, cm, func() error { return nil })
	return err
}

func (a *actuator) ensureWebhookKubeconfig(ctx context.Context, namespace string, checksums map[string]string) error {
	a.logger.Info("Ensuring webhook kubernetes config", "namespace", namespace)
	secret := &corev1.Secret{}
	secret.SetName(config.AuditlogKubecfgSecretName)
	secret.SetNamespace(namespace)
	err := a.client.Get(ctx, client.ObjectKey{Name: config.AuditlogKubecfgSecretName, Namespace: namespace}, secret)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	a.logger.Info("Generating new certificate for webhook kubeconfig", "namespace", namespace)
	cert, err := a.generateCertificate(config.AuditlogProxyServiceName, namespace)
	if err != nil {
		return err
	}

	webhookKubeconfig := clientcmdv1.Config{
		CurrentContext: "auditlog-proxy",
		Contexts: []clientcmdv1.NamedContext{
			{
				Name: "auditlog-proxy",
				Context: clientcmdv1.Context{
					Cluster:  "auditlog-proxy",
					AuthInfo: "auditlog-proxy-auth",
				},
			},
		},
		Clusters: []clientcmdv1.NamedCluster{
			{
				Name: "auditlog-proxy",
				Cluster: clientcmdv1.Cluster{
					Server:                   fmt.Sprintf("https://%s.%s:%d", config.AuditlogProxyServiceName, namespace, 443),
					InsecureSkipTLSVerify:    false,
					CertificateAuthorityData: cert.Cert,
				},
			},
		},
		AuthInfos: []clientcmdv1.NamedAuthInfo{
			{
				Name: "auditlog-proxy-auth",
				AuthInfo: clientcmdv1.AuthInfo{
					Token: "abc",
				},
			},
		},
	}

	rawKubeconfig, err := yaml.Marshal(webhookKubeconfig)
	if err != nil {
		return err
	}

	secret.Data = map[string][]byte{
		"kubeconfig": rawKubeconfig,
		"tls.crt":    cert.Cert,
		"tls.key":    cert.Key,
	}

	checksums[fmt.Sprintf("checksum/configmap-%s", config.AuditlogKubecfgSecretName)] = extensionutil.ComputeChecksum(secret.Data)
	return a.client.Create(ctx, secret)
}

func (a *actuator) ensureKubeAPIServerDeployment(ctx context.Context, namespace string) error {
	dep := &appsv1.Deployment{}
	if err := a.client.Get(ctx, client.ObjectKey{Name: v1beta1constants.DeploymentNameKubeAPIServer, Namespace: namespace}, dep); err != nil {
		return err
	}

	if err := a.controlplaneEnsurer.EnsureKubeAPIServerDeployment(ctx, nil, dep); err != nil {
		return err
	}

	return a.client.Update(ctx, dep)
}

func (a *actuator) generateCertificate(svcName, namespace string) (*Certificate, error) {
	validFor := 5 * 12 * 30 * 24 * time.Hour // ~5 years

	// generate all possible dns hostnames
	dnsSvcSuffix := fmt.Sprintf("%s.%s.svc.cluster.local", svcName, namespace)
	hosts := make([]string, 0)
	host := make([]string, 0)
	for _, seg := range strings.Split(dnsSvcSuffix, ".") {
		host = append(host, seg)
		hosts = append(hosts, strings.Join(host, "."))
	}

	return GenerateRSACertificate(2048, "shoot-auditlog", hosts, time.Now().Add(validFor))
}
