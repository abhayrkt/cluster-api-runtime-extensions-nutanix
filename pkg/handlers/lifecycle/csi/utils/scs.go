// Copyright 2023 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"
	"maps"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/controllers/remote"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nutanix-cloud-native/cluster-api-runtime-extensions-nutanix/api/v1alpha1"
	"github.com/nutanix-cloud-native/cluster-api-runtime-extensions-nutanix/common/pkg/k8s/client"
)

const (
	KindStorageClass        = "StorageClass"
	KindVolumeSnapshotClass = "VolumeSnapshotClass"

	// isDefaultStorageClassAnnotation represents a StorageClass annotation that
	// marks a class as the default StorageClass.
	isDefaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"
)

// volumeSnapshotClassGVK is the GroupVersionKind for the external-snapshotter
// VolumeSnapshotClass resource. We build the object as unstructured because the
// external-snapshotter API types are not vendored and the CRD is installed on
// the workload cluster by the snapshot-controller addon.
var volumeSnapshotClassGVK = schema.GroupVersionKind{
	Group:   "snapshot.storage.k8s.io",
	Version: "v1",
	Kind:    KindVolumeSnapshotClass,
}

var defaultStorageClassMap = map[string]string{
	isDefaultStorageClassAnnotation: "true",
}

func CreateStorageClass(
	providerName string,
	storageClassName string,
	storageClassConfig v1alpha1.StorageClassConfig,
	provisioner v1alpha1.StorageProvisioner,
	isDefault bool,
	defaultParameters map[string]string,
) *storagev1.StorageClass {
	parameters := make(map[string]string, len(defaultParameters)+len(storageClassConfig.Parameters))
	// set the defaults first so that user provided parameters can override them
	maps.Copy(parameters, defaultParameters)
	// set user provided parameters, overriding any defaults with the same key
	maps.Copy(parameters, storageClassConfig.Parameters)

	sc := storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindStorageClass,
			APIVersion: storagev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: providerName + "-" + storageClassName,
		},
		Provisioner:          string(provisioner),
		Parameters:           parameters,
		VolumeBindingMode:    storageClassConfig.VolumeBindingMode,
		ReclaimPolicy:        storageClassConfig.ReclaimPolicy,
		AllowVolumeExpansion: ptr.To(storageClassConfig.AllowExpansion),
	}
	if isDefault {
		sc.Annotations = defaultStorageClassMap
	}
	return &sc
}

func CreateStorageClassesOnRemote(
	ctx context.Context,
	cl ctrlclient.Client,
	configs map[string]v1alpha1.StorageClassConfig,
	cluster *clusterv1.Cluster,
	defaultStorage v1alpha1.DefaultStorage,
	csiProvider string,
	provisioner v1alpha1.StorageProvisioner,
	defaultParameters map[string]string,
) error {
	remoteClient, err := remote.NewClusterClient(
		ctx,
		"",
		cl,
		ctrlclient.ObjectKeyFromObject(cluster),
	)
	if err != nil {
		return fmt.Errorf("error creating client for remote cluster: %w", err)
	}

	for name, config := range configs {
		setAsDefault := csiProvider == defaultStorage.Provider &&
			name == defaultStorage.StorageClassConfig
		sc := CreateStorageClass(
			csiProvider,
			name,
			config,
			provisioner,
			setAsDefault,
			defaultParameters,
		)
		if err := client.ServerSideApply(ctx, remoteClient, sc, client.ForceOwnership); err != nil {
			return fmt.Errorf("error creating storage class %v on remote cluster: %w", sc, err)
		}
	}

	return nil
}

// CreateVolumeSnapshotClass builds a VolumeSnapshotClass object as unstructured.
// The external-snapshotter API types are not vendored, so we set the GVK and
// fields directly. deletionPolicy is set to "Delete" so the underlying snapshot
// is removed when the VolumeSnapshot is deleted.
func CreateVolumeSnapshotClass(
	name string,
	driver string,
	parameters map[string]string,
) *unstructured.Unstructured {
	vsc := &unstructured.Unstructured{}
	vsc.SetGroupVersionKind(volumeSnapshotClassGVK)
	vsc.SetName(name)
	vsc.Object["driver"] = driver
	vsc.Object["deletionPolicy"] = "Delete"
	if len(parameters) > 0 {
		params := make(map[string]interface{}, len(parameters))
		for k, v := range parameters {
			params[k] = v
		}
		vsc.Object["parameters"] = params
	}
	return vsc
}

// CreateVolumeSnapshotClassOnRemote applies the given VolumeSnapshotClass to the
// remote (workload) cluster. This ensures the class exists on every cluster the
// CSI handler manages, including the self-managed management cluster, rather than
// relying on the upstream CSI Helm chart which only creates it when volumeClass
// is enabled.
func CreateVolumeSnapshotClassOnRemote(
	ctx context.Context,
	cl ctrlclient.Client,
	cluster *clusterv1.Cluster,
	name string,
	driver string,
	parameters map[string]string,
) error {
	remoteClient, err := remote.NewClusterClient(
		ctx,
		"",
		cl,
		ctrlclient.ObjectKeyFromObject(cluster),
	)
	if err != nil {
		return fmt.Errorf("error creating client for remote cluster: %w", err)
	}

	vsc := CreateVolumeSnapshotClass(name, driver, parameters)
	if err := client.ServerSideApply(ctx, remoteClient, vsc, client.ForceOwnership); err != nil {
		// The VolumeSnapshotClass CRD is installed by the optional
		// snapshot-controller addon. When that addon is not enabled, the CRD is
		// absent and the RESTMapper returns a NoMatchError. Treat this as a skip
		// rather than a failure so omitting the snapshot-controller addon does
		// not block cluster provisioning.
		if meta.IsNoMatchError(err) {
			ctrl.LoggerFrom(ctx).V(5).Info(
				"Skipping VolumeSnapshotClass creation: CRD not installed "+
					"(snapshot-controller addon is not enabled)",
				"name", name,
			)
			return nil
		}
		return fmt.Errorf(
			"error creating VolumeSnapshotClass %q on remote cluster: %w",
			name,
			err,
		)
	}

	return nil
}
