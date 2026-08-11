// Copyright 2023 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"

	"github.com/nutanix-cloud-native/cluster-api-runtime-extensions-nutanix/api/v1alpha1"
)

var (
	defaultParameters = map[string]string{
		"csi.storage.k8s.io/fstype": "ext4",
		"type":                      "gp3",
	}
	userProviderParameters = map[string]string{
		"csi.storage.k8s.io/fstype": "xfs",
		"flashMode":                 "ENABLED",
		"storageContainer":          "storage-container-name",
		"chapAuth":                  "ENABLED",
		"storageType":               "NutanixVolumes",
		"whitelistIPMode":           "ENABLED",
		"whitelistIPAddr":           "1.1.1.1",
	}

	combinedParameters = map[string]string{
		"csi.storage.k8s.io/fstype": "xfs",
		"type":                      "gp3",
		"flashMode":                 "ENABLED",
		"storageContainer":          "storage-container-name",
		"chapAuth":                  "ENABLED",
		"storageType":               "NutanixVolumes",
		"whitelistIPMode":           "ENABLED",
		"whitelistIPAddr":           "1.1.1.1",
	}
)

func TestCreateStorageClass(t *testing.T) {
	const (
		testProviderName = "test-provider"
		testSCName       = "test-sc"
	)

	tests := []struct {
		name                 string
		storageConfig        v1alpha1.StorageClassConfig
		provisioner          v1alpha1.StorageProvisioner
		setAsDefault         bool
		defaultParameters    map[string]string
		expectedStorageClass *storagev1.StorageClass
	}{
		{
			name: "with only default parameters",
			storageConfig: v1alpha1.StorageClassConfig{
				ReclaimPolicy:     ptr.To(v1alpha1.VolumeReclaimDelete),
				VolumeBindingMode: ptr.To(v1alpha1.VolumeBindingWaitForFirstConsumer),
				Parameters:        nil,
				AllowExpansion:    true,
			},
			provisioner:       v1alpha1.AWSEBSProvisioner,
			defaultParameters: defaultParameters,
			expectedStorageClass: &storagev1.StorageClass{
				TypeMeta: metav1.TypeMeta{
					Kind:       KindStorageClass,
					APIVersion: storagev1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: testProviderName + "-" + testSCName,
				},
				Parameters:           defaultParameters,
				ReclaimPolicy:        ptr.To(corev1.PersistentVolumeReclaimDelete),
				VolumeBindingMode:    ptr.To(storagev1.VolumeBindingWaitForFirstConsumer),
				Provisioner:          string(v1alpha1.AWSEBSProvisioner),
				AllowVolumeExpansion: ptr.To(true),
			},
		},
		{
			name: "with only user provided parameters",
			storageConfig: v1alpha1.StorageClassConfig{
				ReclaimPolicy:     ptr.To(v1alpha1.VolumeReclaimDelete),
				VolumeBindingMode: ptr.To(v1alpha1.VolumeBindingWaitForFirstConsumer),
				Parameters:        userProviderParameters,
			},
			provisioner: v1alpha1.NutanixProvisioner,
			expectedStorageClass: &storagev1.StorageClass{
				TypeMeta: metav1.TypeMeta{
					Kind:       KindStorageClass,
					APIVersion: storagev1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: testProviderName + "-" + testSCName,
				},
				Parameters:           userProviderParameters,
				ReclaimPolicy:        ptr.To(corev1.PersistentVolumeReclaimDelete),
				VolumeBindingMode:    ptr.To(storagev1.VolumeBindingWaitForFirstConsumer),
				Provisioner:          string(v1alpha1.NutanixProvisioner),
				AllowVolumeExpansion: ptr.To(false),
			},
		},
		{
			name: "with both default and user provided parameters",
			storageConfig: v1alpha1.StorageClassConfig{
				ReclaimPolicy:     ptr.To(v1alpha1.VolumeReclaimDelete),
				VolumeBindingMode: ptr.To(v1alpha1.VolumeBindingWaitForFirstConsumer),
				Parameters:        userProviderParameters,
				AllowExpansion:    true,
			},
			provisioner:       v1alpha1.AWSEBSProvisioner,
			defaultParameters: defaultParameters,
			expectedStorageClass: &storagev1.StorageClass{
				TypeMeta: metav1.TypeMeta{
					Kind:       KindStorageClass,
					APIVersion: storagev1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: testProviderName + "-" + testSCName,
				},
				Parameters:           combinedParameters,
				ReclaimPolicy:        ptr.To(corev1.PersistentVolumeReclaimDelete),
				VolumeBindingMode:    ptr.To(storagev1.VolumeBindingWaitForFirstConsumer),
				Provisioner:          string(v1alpha1.AWSEBSProvisioner),
				AllowVolumeExpansion: ptr.To(true),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := CreateStorageClass(
				testProviderName,
				testSCName,
				tt.storageConfig,
				tt.provisioner,
				tt.setAsDefault,
				tt.defaultParameters,
			)
			if diff := cmp.Diff(sc, tt.expectedStorageClass); diff != "" {
				t.Errorf("CreateStorageClass() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateVolumeSnapshotClass(t *testing.T) {
	tests := []struct {
		name       string
		vscName    string
		driver     string
		parameters map[string]string
		expected   *unstructured.Unstructured
	}{
		{
			name:    "with parameters",
			vscName: "nutanix-snapshot-class",
			driver:  "csi.nutanix.com",
			parameters: map[string]string{
				"storageType": "NutanixVolumes",
				"csi.storage.k8s.io/snapshotter-secret-name": "nutanix-csi-credentials",
			},
			expected: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "snapshot.storage.k8s.io/v1",
					"kind":       "VolumeSnapshotClass",
					"metadata": map[string]interface{}{
						"name": "nutanix-snapshot-class",
					},
					"driver":         "csi.nutanix.com",
					"deletionPolicy": "Delete",
					"parameters": map[string]interface{}{
						"storageType": "NutanixVolumes",
						"csi.storage.k8s.io/snapshotter-secret-name": "nutanix-csi-credentials",
					},
				},
			},
		},
		{
			name:       "without parameters",
			vscName:    "nutanix-snapshot-class",
			driver:     "csi.nutanix.com",
			parameters: nil,
			expected: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "snapshot.storage.k8s.io/v1",
					"kind":       "VolumeSnapshotClass",
					"metadata": map[string]interface{}{
						"name": "nutanix-snapshot-class",
					},
					"driver":         "csi.nutanix.com",
					"deletionPolicy": "Delete",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vsc := CreateVolumeSnapshotClass(tt.vscName, tt.driver, tt.parameters)
			if diff := cmp.Diff(tt.expected, vsc); diff != "" {
				t.Errorf("CreateVolumeSnapshotClass() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestVolumeSnapshotClassGVKUnmappedIsNoMatchError guards the assumption the
// missing-CRD skip in CreateVolumeSnapshotClassOnRemote relies on: when the
// optional snapshot-controller addon is omitted, the VolumeSnapshotClass CRD is
// not installed, so the cluster's RESTMapper cannot resolve the GVK. The real
// controller-runtime client resolves the GVK via RESTMapping before issuing the
// apply (see clientRestResources.getResource), and an unmapped GVK yields a
// NoMatchError, which the handler treats as a skip rather than a failure.
func TestVolumeSnapshotClassGVKUnmappedIsNoMatchError(t *testing.T) {
	// An empty RESTMapper mirrors a cluster where the VolumeSnapshotClass CRD has
	// not been installed.
	restMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{})

	vsc := CreateVolumeSnapshotClass(
		"nutanix-snapshot-class",
		"csi.nutanix.com",
		map[string]string{"storageType": "NutanixVolumes"},
	)
	gvk := vsc.GroupVersionKind()

	_, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err == nil {
		t.Fatal("expected a RESTMapping error for the unmapped VolumeSnapshotClass GVK, got nil")
	}
	if !meta.IsNoMatchError(err) {
		t.Errorf("expected a NoMatchError, got: %v", err)
	}
}
