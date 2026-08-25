// Copyright 2026 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

	"github.com/nutanix-cloud-native/cluster-api-runtime-extensions-nutanix/api/v1alpha1"
	"github.com/nutanix-cloud-native/cluster-api-runtime-extensions-nutanix/api/variables"
)

func TestIsMetroFailureDomain(t *testing.T) {
	t.Parallel()

	assert.True(t, IsMetroFailureDomain(MetroFailureDomainPrefix+"metro-1"))
	assert.True(t, IsMetroFailureDomain(MetroSiteFailureDomainPrefix+"site-1"))
	assert.False(t, IsMetroFailureDomain("fd-1"))
	assert.False(t, IsMetroFailureDomain(""))
}

func TestIsMetroCluster(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cluster  *clusterv1.Cluster
		expected bool
	}{
		{
			name:     "nil cluster",
			cluster:  nil,
			expected: false,
		},
		{
			name: "no topology",
			cluster: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "no-topology"},
			},
			expected: false,
		},
		{
			name: "control plane metro failure domain",
			cluster: clusterWithConfigAndWorkers(
				&variables.ClusterConfigSpec{
					ControlPlane: &variables.ControlPlaneSpec{
						Nutanix: &v1alpha1.NutanixControlPlaneNodeSpec{
							FailureDomains: []string{MetroFailureDomainPrefix + "metro-1"},
						},
					},
				},
				"",
			),
			expected: true,
		},
		{
			name: "worker metro failure domain",
			cluster: clusterWithConfigAndWorkers(
				&variables.ClusterConfigSpec{},
				MetroSiteFailureDomainPrefix+"site-1",
			),
			expected: true,
		},
		{
			name: "non-metro failure domains",
			cluster: clusterWithConfigAndWorkers(
				&variables.ClusterConfigSpec{
					ControlPlane: &variables.ControlPlaneSpec{
						Nutanix: &v1alpha1.NutanixControlPlaneNodeSpec{
							FailureDomains: []string{"pe-fd-1"},
						},
					},
				},
				"pe-fd-2",
			),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsMetroCluster(tt.cluster))
		})
	}
}

func clusterWithConfigAndWorkers(
	cfg *variables.ClusterConfigSpec,
	workerFD string,
) *clusterv1.Cluster {
	raw, err := json.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: clusterv1.ClusterSpec{
			Topology: clusterv1.Topology{
				ClassRef: clusterv1.ClusterClassRef{Name: "dummy-class"},
				Version:  "v1.30.0",
				Variables: []clusterv1.ClusterVariable{{
					Name:  v1alpha1.ClusterConfigVariableName,
					Value: apiextensionsv1.JSON{Raw: raw},
				}},
			},
		},
	}
	if workerFD != "" {
		cluster.Spec.Topology.Workers.MachineDeployments = []clusterv1.MachineDeploymentTopology{{
			Name:          "md-0",
			Class:         "default-worker",
			FailureDomain: workerFD,
		}}
	}
	return cluster
}
