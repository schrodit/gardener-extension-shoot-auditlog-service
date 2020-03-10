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
	"context"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/apis/config"
	extensionswebhook "github.com/gardener/gardener-extensions/pkg/webhook"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane/genericmutator"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewEnsurer creates a new controlplane ensurer.
func NewEnsurer(logger logr.Logger) genericmutator.Ensurer {
	return &ensurer{
		logger: logger.WithName("shoot-auditlog-service-ensurer"),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	client client.Client
	logger logr.Logger
}

// InjectClient injects the given client into the ensurer.
func (e *ensurer) InjectClient(client client.Client) error {
	e.client = client
	return nil
}

func (e *ensurer) EnsureKubeAPIServerDeployment(ctx context.Context, _ genericmutator.EnsurerContext, dep *appsv1.Deployment) error {
	e.logger.Info("Ensuring apiserver deployment")

	// do not mutate the apiserver if the secrets are missing
	secret := &corev1.Secret{}
	if err := e.client.Get(ctx, client.ObjectKey{Name: config.AuditlogKubecfgSecretName, Namespace: dep.Namespace}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	cm := &corev1.ConfigMap{}
	if err := e.client.Get(ctx, client.ObjectKey{Name: config.AuditlogPolicyConfigMapName, Namespace: dep.Namespace}, cm); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	template := &dep.Spec.Template
	ps := &template.Spec

	if c := extensionswebhook.ContainerWithName(ps.Containers, "kube-apiserver"); c != nil {
		ensureKubeAPIServerCommandLineArgs(c)
		ensureVolumeMounts(c)
	}
	ensureVolumes(ps)
	ensureNetworkPolicyLabel(template)

	if err := controlplane.EnsureSecretChecksumAnnotation(ctx, template, e.client, dep.Namespace, config.AuditlogKubecfgSecretName); err != nil {
		return err
	}
	return controlplane.EnsureConfigMapChecksumAnnotation(ctx, template, e.client, dep.Namespace, config.AuditlogPolicyConfigMapName)
}

var (
	auditlogWebhookKubeconfigMount = corev1.VolumeMount{
		Name:      config.AuditlogKubecfgSecretName,
		MountPath: "/etc/kube-apiserver/auditwebhook",
	}

	auditlogPolicyConfigMount = corev1.VolumeMount{
		Name:      config.AuditlogPolicyConfigMapName,
		MountPath: "/etc/kube-apiserver/audit",
	}

	auditlogWebhookKubeconfigVolume = corev1.Volume{
		Name: config.AuditlogKubecfgSecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: config.AuditlogKubecfgSecretName,
			},
		},
	}

	auditlogPolicyConfigVolume = corev1.Volume{
		Name: config.AuditlogPolicyConfigMapName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: config.AuditlogPolicyConfigMapName},
			},
		},
	}
)

func ensureKubeAPIServerCommandLineArgs(c *corev1.Container) {
	c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--audit-webhook-config-file=",
		"/etc/kube-apiserver/auditwebhook/kubeconfig")
	c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--audit-policy-file=",
		"/etc/kube-apiserver/audit/audit-policy.yaml")
}

func ensureVolumeMounts(c *corev1.Container) {
	c.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(c.VolumeMounts, auditlogWebhookKubeconfigMount)
	c.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(c.VolumeMounts, auditlogPolicyConfigMount)
}

func ensureVolumes(ps *corev1.PodSpec) {
	ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, auditlogWebhookKubeconfigVolume)
	ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, auditlogPolicyConfigVolume)
}

func ensureNetworkPolicyLabel(template *corev1.PodTemplateSpec) {
	if template.Labels == nil {
		template.Labels = map[string]string{}
	}
	template.Labels[config.AllowAuditlogProxyNetworkPolicyLabel] = "allow"
}
