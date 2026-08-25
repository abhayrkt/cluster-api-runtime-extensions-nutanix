// Copyright 2026 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"strings"

	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

	"github.com/nutanix-cloud-native/cluster-api-runtime-extensions-nutanix/api/variables"
)

const (
	// MetroFailureDomainPrefix is the CAPI failure-domain name prefix for a NutanixMetro.
	MetroFailureDomainPrefix = "NutanixMetro/"
	// MetroSiteFailureDomainPrefix is the CAPI failure-domain name prefix for a NutanixMetroSite.
	MetroSiteFailureDomainPrefix = "NutanixMetroSite/"
)

// IsMetroFailureDomain reports whether fd names a NutanixMetro or NutanixMetroSite.
func IsMetroFailureDomain(fd string) bool {
	return strings.HasPrefix(fd, MetroFailureDomainPrefix) ||
		strings.HasPrefix(fd, MetroSiteFailureDomainPrefix)
}

// IsMetroCluster returns true when the cluster uses metro-aware failure domains,
// i.e. any control-plane or worker failure domain references a NutanixMetro or
// NutanixMetroSite object (identified by the respective name prefix).
func IsMetroCluster(cluster *clusterv1.Cluster) bool {
	if cluster == nil || !cluster.Spec.Topology.IsDefined() {
		return false
	}

	clusterConfig, err := variables.UnmarshalClusterConfigVariable(cluster.Spec.Topology.Variables)
	if err == nil &&
		clusterConfig != nil &&
		clusterConfig.ControlPlane != nil &&
		clusterConfig.ControlPlane.Nutanix != nil {
		for _, fd := range clusterConfig.ControlPlane.Nutanix.FailureDomains {
			if IsMetroFailureDomain(fd) {
				return true
			}
		}
	}

	for i := range cluster.Spec.Topology.Workers.MachineDeployments {
		if IsMetroFailureDomain(cluster.Spec.Topology.Workers.MachineDeployments[i].FailureDomain) {
			return true
		}
	}

	return false
}
