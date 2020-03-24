// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package shoot

import (
	"context"
	"fmt"
	"strings"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardencoreinformers "github.com/gardener/gardener/pkg/client/core/informers/externalversions/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/gardener/pkg/operation/garden"
	"github.com/gardener/gardener/pkg/utils"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	"github.com/Masterminds/semver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// New takes a <k8sGardenClient>, the <k8sGardenCoreInformers> and a <shoot> manifest, and creates a new Shoot representation.
// It will add the CloudProfile, the cloud provider secret, compute the internal cluster domain and identify the cloud provider.
func New(k8sGardenClient kubernetes.Interface, k8sGardenCoreInformers gardencoreinformers.Interface, shoot *gardencorev1beta1.Shoot, projectName string, disableDNS bool, internalDomain *garden.Domain, defaultDomains []*garden.Domain) (*Shoot, error) {
	var (
		secret *corev1.Secret
		err    error
	)

	cloudProfile, err := k8sGardenCoreInformers.CloudProfiles().Lister().Get(shoot.Spec.CloudProfileName)
	if err != nil {
		return nil, err
	}

	binding, err := k8sGardenCoreInformers.SecretBindings().Lister().SecretBindings(shoot.Namespace).Get(shoot.Spec.SecretBindingName)
	if err != nil {
		return nil, err
	}
	secret = &corev1.Secret{}
	if err = k8sGardenClient.Client().Get(context.TODO(), kutil.Key(binding.SecretRef.Namespace, binding.SecretRef.Name), secret); err != nil {
		return nil, err
	}

	seedNamespace := ComputeTechnicalID(projectName, shoot)

	extensions, err := calculateExtensions(k8sGardenClient.Client(), shoot, seedNamespace)
	if err != nil {
		return nil, fmt.Errorf("Cannot calculate required extensions for shoot %s: %v", shoot.Name, err)
	}

	shootObj := &Shoot{
		Info:         shoot,
		Secret:       secret,
		CloudProfile: cloudProfile,

		SeedNamespace: seedNamespace,

		DisableDNS:            disableDNS,
		InternalClusterDomain: ConstructInternalClusterDomain(shoot.Name, projectName, internalDomain),
		ExternalClusterDomain: ConstructExternalClusterDomain(shoot),

		HibernationEnabled:     gardencorev1beta1helper.HibernationIsEnabled(shoot),
		WantsClusterAutoscaler: false,

		Extensions: extensions,
	}
	shootObj.OperatingSystemConfigsMap = make(map[string]OperatingSystemConfigs, len(shootObj.GetWorkerNames()))

	// Determine information about external domain for shoot cluster.
	externalDomain, err := ConstructExternalDomain(context.TODO(), k8sGardenClient.Client(), shoot, secret, defaultDomains)
	if err != nil && !(IsIncompleteDNSConfigError(err) && shoot.DeletionTimestamp != nil && len(shoot.Status.UID) == 0) {
		return nil, err
	}
	shootObj.ExternalDomain = externalDomain

	// Store the Kubernetes version in the format <major>.<minor> on the Shoot object.
	v, err := semver.NewVersion(shoot.Spec.Kubernetes.Version)
	if err != nil {
		return nil, err
	}
	shootObj.KubernetesMajorMinorVersion = fmt.Sprintf("%d.%d", v.Major(), v.Minor())

	needsAutoscaler, err := gardencorev1beta1helper.ShootWantsClusterAutoscaler(shoot)
	if err != nil {
		return nil, err
	}
	shootObj.WantsClusterAutoscaler = needsAutoscaler

	return shootObj, nil
}

func calculateExtensions(gardenClient client.Client, shoot *gardencorev1beta1.Shoot, seedNamespace string) (map[string]Extension, error) {
	var controllerRegistrations = &gardencorev1beta1.ControllerRegistrationList{}
	if err := gardenClient.List(context.TODO(), controllerRegistrations); err != nil {
		return nil, err
	}
	return MergeExtensions(controllerRegistrations.Items, shoot.Spec.Extensions, seedNamespace)
}

// GetIngressFQDN returns the fully qualified domain name of ingress sub-resource for the Shoot cluster. The
// end result is '<subDomain>.<ingressPrefix>.<clusterDomain>'.
func (s *Shoot) GetIngressFQDN(subDomain string) string {
	if s.Info.Spec.DNS == nil || s.Info.Spec.DNS.Domain == nil {
		return ""
	}
	return fmt.Sprintf("%s.%s.%s", subDomain, common.IngressPrefix, *(s.Info.Spec.DNS.Domain))
}

// GetPurpose returns the purpose of the shoot or 'evaluation' if it's nil.
func (s *Shoot) GetPurpose() gardencorev1beta1.ShootPurpose {
	if v := s.Info.Spec.Purpose; v != nil {
		return *v
	}
	return gardencorev1beta1.ShootPurposeEvaluation
}

// GetWorkerNames returns a list of names of the worker groups in the Shoot manifest.
func (s *Shoot) GetWorkerNames() []string {
	var workerNames []string
	for _, worker := range s.Info.Spec.Provider.Workers {
		workerNames = append(workerNames, worker.Name)
	}
	return workerNames
}

// GetNodeCount returns the sum of all 'maximum' fields of all worker groups of the Shoot.
func (s *Shoot) GetNodeCount() int32 {
	var nodeCount int32
	for _, worker := range s.Info.Spec.Provider.Workers {
		nodeCount += worker.Maximum
	}
	return nodeCount
}

// GetPodNetwork returns the pod network CIDR for the Shoot cluster. If no CIDR was specified then an
// empty string is returned.
func (s *Shoot) GetPodNetwork() string {
	if val := s.Info.Spec.Networking.Pods; val != nil {
		return *val
	}
	return ""
}

// GetServiceNetwork returns the service network CIDR for the Shoot cluster. If no CIDR was specified then an
// empty string is returned.
func (s *Shoot) GetServiceNetwork() string {
	if val := s.Info.Spec.Networking.Services; val != nil {
		return *val
	}
	return ""
}

// GetNodeNetwork returns the nodes network CIDR for the Shoot cluster. If the infrastructure extension
// controller has generated a nodes network then this CIDR will take priority. Otherwise, the nodes network
// CIDR specified in the shoot will be returned (if possible). If no CIDR was specified then nil is returned.
func (s *Shoot) GetNodeNetwork() *string {
	if val := s.Info.Spec.Networking.Nodes; val != nil {
		return val
	}
	return nil
}

// KubernetesDashboardEnabled returns true if the kubernetes-dashboard addon is enabled in the Shoot manifest.
func (s *Shoot) KubernetesDashboardEnabled() bool {
	return s.Info.Spec.Addons != nil && s.Info.Spec.Addons.KubernetesDashboard != nil && s.Info.Spec.Addons.KubernetesDashboard.Enabled
}

// NginxIngressEnabled returns true if the nginx-ingress addon is enabled in the Shoot manifest.
func (s *Shoot) NginxIngressEnabled() bool {
	return s.Info.Spec.Addons != nil && s.Info.Spec.Addons.NginxIngress != nil && s.Info.Spec.Addons.NginxIngress.Enabled
}

// ComputeCloudConfigSecretName computes the name for a secret which contains the original cloud config for
// the worker group with the given <workerName>. It is build by the cloud config secret prefix, the worker
// name itself and a hash of the minor Kubernetes version of the Shoot cluster.
func (s *Shoot) ComputeCloudConfigSecretName(workerName string) string {
	return fmt.Sprintf("%s-%s-%s", common.CloudConfigPrefix, workerName, utils.ComputeSHA256Hex([]byte(s.KubernetesMajorMinorVersion))[:5])
}

// GetReplicas returns the given <wokenUp> number if the shoot is not hibernated, or zero otherwise.
func (s *Shoot) GetReplicas(wokenUp int) int {
	if s.HibernationEnabled {
		return 0
	}
	return wokenUp
}

// ComputeAPIServerURL takes a boolean value identifying whether the component connecting to the API server
// runs in the Seed cluster <runsInSeed>, and a boolean value <useInternalClusterDomain> which determines whether the
// internal or the external cluster domain should be used.
func (s *Shoot) ComputeAPIServerURL(runsInSeed, useInternalClusterDomain bool, apiServerAddress string) string {
	if runsInSeed {
		return v1beta1constants.DeploymentNameKubeAPIServer
	}

	if s.DisableDNS {
		return apiServerAddress
	}

	if gardencorev1beta1helper.ShootUsesUnmanagedDNS(s.Info) {
		return common.GetAPIServerDomain(s.InternalClusterDomain)
	}

	if useInternalClusterDomain {
		return common.GetAPIServerDomain(s.InternalClusterDomain)
	}

	return common.GetAPIServerDomain(*s.ExternalClusterDomain)
}

// IPVSEnabled returns true if IPVS is enabled for the shoot.
func (s *Shoot) IPVSEnabled() bool {
	return s.Info.Spec.Kubernetes.KubeProxy != nil &&
		s.Info.Spec.Kubernetes.KubeProxy.Mode != nil &&
		*s.Info.Spec.Kubernetes.KubeProxy.Mode == gardencorev1beta1.ProxyModeIPVS
}

// TechnicalIDPrefix is a prefix used for a shoot's technical id.
const TechnicalIDPrefix = "shoot--"

// ComputeTechnicalID determines the technical id of that Shoot which is later used for the name of the
// namespace and for tagging all the resources created in the infrastructure.
func ComputeTechnicalID(projectName string, shoot *gardencorev1beta1.Shoot) string {
	// Use the stored technical ID in the Shoot's status field if it's there.
	// For backwards compatibility we keep the pattern as it was before we had to change it
	// (double hyphens).
	if len(shoot.Status.TechnicalID) > 0 {
		return shoot.Status.TechnicalID
	}

	// New clusters shall be created with the new technical id (double hyphens).
	return fmt.Sprintf("%s%s--%s", TechnicalIDPrefix, projectName, shoot.Name)
}

// UnfoldTechnicalID extracts the project and shoot names from Shoot's TechnicalID which is also used
// for namespace
func UnfoldTechnicalID(technicalID string) (projectName, shootName string) {
	if len(technicalID) == 0 {
		return
	}

	var technicalIDComponents []string
	if strings.HasPrefix(technicalID, "shoot--") {
		technicalIDComponents = strings.Split(technicalID, "--")
	} else if strings.HasPrefix(technicalID, "shoot-") {
		technicalIDComponents = strings.Split(technicalID, "-")
	} else {
		return
	}

	projectName = technicalIDComponents[1]
	shootName = technicalIDComponents[2]
	return
}

// ConstructInternalClusterDomain constructs the internal base domain pof this shoot cluster.
// It is only used for internal purposes (all kubeconfigs except the one which is received by the
// user will only talk with the kube-apiserver via a DNS record of domain). In case the given <internalDomain>
// already contains "internal", the result is constructed as "<shootName>.<shootProject>.<internalDomain>."
// In case it does not, the word "internal" will be appended, resulting in
// "<shootName>.<shootProject>.internal.<internalDomain>".
func ConstructInternalClusterDomain(shootName, shootProject string, internalDomain *garden.Domain) string {
	if internalDomain == nil {
		return ""
	}
	if strings.Contains(internalDomain.Domain, common.InternalDomainKey) {
		return fmt.Sprintf("%s.%s.%s", shootName, shootProject, internalDomain.Domain)
	}
	return fmt.Sprintf("%s.%s.%s.%s", shootName, shootProject, common.InternalDomainKey, internalDomain.Domain)
}

// ConstructExternalClusterDomain constructs the external Shoot cluster domain, i.e. the domain which will be put
// into the Kubeconfig handed out to the user.
func ConstructExternalClusterDomain(shoot *gardencorev1beta1.Shoot) *string {
	if shoot.Spec.DNS == nil || shoot.Spec.DNS.Domain == nil {
		return nil
	}
	return shoot.Spec.DNS.Domain
}

// ConstructExternalDomain constructs an object containing all relevant information of the external domain that
// shall be used for a shoot cluster - based on the configuration of the Garden cluster and the shoot itself.
func ConstructExternalDomain(ctx context.Context, client client.Client, shoot *gardencorev1beta1.Shoot, shootSecret *corev1.Secret, defaultDomains []*garden.Domain) (*garden.Domain, error) {
	externalClusterDomain := ConstructExternalClusterDomain(shoot)
	if externalClusterDomain == nil {
		return nil, nil
	}

	var (
		externalDomain = &garden.Domain{Domain: *shoot.Spec.DNS.Domain}
		defaultDomain  = garden.DomainIsDefaultDomain(*externalClusterDomain, defaultDomains)
	)

	switch {
	case len(shoot.Spec.DNS.Providers) > 0 && shoot.Spec.DNS.Providers[0].SecretName != nil:
		secret := &corev1.Secret{}
		if err := client.Get(ctx, kutil.Key(shoot.Namespace, *shoot.Spec.DNS.Providers[0].SecretName), secret); err != nil {
			return nil, fmt.Errorf("could not get dns provider secret \"%s\": %+v", *shoot.Spec.DNS.Providers[0].SecretName, err)
		}
		externalDomain.SecretData = secret.Data
		externalDomain.Provider = *shoot.Spec.DNS.Providers[0].Type
		if domains := shoot.Spec.DNS.Providers[0].Domains; domains != nil {
			externalDomain.IncludeDomains = domains.Include
			externalDomain.ExcludeDomains = domains.Exclude
		}
		if zones := shoot.Spec.DNS.Providers[0].Zones; zones != nil {
			externalDomain.IncludeZones = zones.Include
			externalDomain.ExcludeZones = zones.Exclude
		}

	case defaultDomain != nil:
		externalDomain.SecretData = defaultDomain.SecretData
		externalDomain.Provider = defaultDomain.Provider
		externalDomain.IncludeDomains = defaultDomain.IncludeDomains
		externalDomain.ExcludeDomains = defaultDomain.ExcludeDomains
		externalDomain.IncludeZones = defaultDomain.IncludeZones
		externalDomain.ExcludeZones = defaultDomain.ExcludeZones

	case len(shoot.Spec.DNS.Providers) > 0 && shoot.Spec.DNS.Providers[0].SecretName == nil:
		externalDomain.SecretData = shootSecret.Data
		externalDomain.Provider = *shoot.Spec.DNS.Providers[0].Type
		if domains := shoot.Spec.DNS.Providers[0].Domains; domains != nil {
			externalDomain.IncludeDomains = domains.Include
			externalDomain.ExcludeDomains = domains.Exclude
		}
		if zones := shoot.Spec.DNS.Providers[0].Zones; zones != nil {
			externalDomain.IncludeZones = zones.Include
			externalDomain.ExcludeZones = zones.Exclude
		}

	default:
		return nil, &IncompleteDNSConfigError{}
	}

	return externalDomain, nil
}

// ExtensionDefaultTimeout is the default timeout and defines how long Gardener should wait
// for a successful reconciliation of this extension resource.
const ExtensionDefaultTimeout = 3 * time.Minute

// MergeExtensions merges the given controller registrations with the given extensions, expecting that each type in extensions is also represented in the registration.
func MergeExtensions(registrations []gardencorev1beta1.ControllerRegistration, extensions []gardencorev1beta1.Extension, namespace string) (map[string]Extension, error) {
	var (
		typeToExtension    = make(map[string]Extension)
		requiredExtensions = make(map[string]Extension)
	)
	// Extensions enabled by default for all Shoot clusters.
	for _, reg := range registrations {
		for _, res := range reg.Spec.Resources {
			if res.Kind != extensionsv1alpha1.ExtensionResource {
				continue
			}

			var timeout time.Duration
			if res.ReconcileTimeout != nil {
				timeout = res.ReconcileTimeout.Duration
			} else {
				timeout = ExtensionDefaultTimeout
			}

			typeToExtension[res.Type] = Extension{
				Extension: extensionsv1alpha1.Extension{
					ObjectMeta: metav1.ObjectMeta{
						Name:      res.Type,
						Namespace: namespace,
					},
					Spec: extensionsv1alpha1.ExtensionSpec{
						DefaultSpec: extensionsv1alpha1.DefaultSpec{
							Type: res.Type,
						},
					},
				},
				Timeout: timeout,
			}

			if res.GloballyEnabled != nil && *res.GloballyEnabled {
				requiredExtensions[res.Type] = typeToExtension[res.Type]
			}
		}
	}

	// Extensions defined in Shoot resource.
	for _, extension := range extensions {
		obj, ok := typeToExtension[extension.Type]
		if ok {
			if extension.ProviderConfig != nil {
				providerConfig := extension.ProviderConfig.RawExtension
				obj.Spec.ProviderConfig = &providerConfig
			}
			requiredExtensions[extension.Type] = obj
			continue
		}
	}

	return requiredExtensions, nil
}
