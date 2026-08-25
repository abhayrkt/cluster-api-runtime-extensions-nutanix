<!--
 Copyright 2026 Nutanix. All rights reserved.
 SPDX-License-Identifier: Apache-2.0
-->

# Feature Specification: Default and validate CSI computeAffinity for metro clusters

**Jira Ticket**: N/A (no ticket provided)
**Feature Branch**: `cursor/metro-csi-compute-affinity-43d1`
**Created**: 2026-08-25
**Status**: Draft
**Input**: User description: "Add an admission webhook to add computeAffinity: DISABLED to storageClass if the cluster is a metro cluster. If user has provided the flag computeAffinity explicitly and the value is other than DISABLED then fail saying not supported for the metro cluster."

## Background

Nutanix CSI StorageClass parameter `computeAffinity` pins a volume to the Prism Element that hosts the consuming VM when set to a value other than `DISABLED`. Metro clusters span two Prism Elements with synchronous replication. Volume-to-host PE affinity is incompatible with that topology, so metro clusters must use `computeAffinity: DISABLED`.

CAREN already detects metro clusters from NutanixMetro / NutanixMetroSite failure-domain prefixes (used today to enable CSI MPIO Helm values). This feature applies the same detection at Cluster admission time so every Nutanix CSI StorageClassConfig on a metro Cluster is forced to `DISABLED`, and explicit incompatible values are rejected before the Cluster is persisted.

## User Scenarios & Testing

### User Story 1 - Metro cluster gets computeAffinity DISABLED by default (Priority: P1)

An operator creates a Nutanix Cluster that uses metro failure domains and Nutanix CSI StorageClassConfigs without setting `computeAffinity`. Admission should persist `computeAffinity: DISABLED` on each Nutanix CSI StorageClassConfig so the CSI lifecycle handler creates StorageClasses that work on metro.

**Why this priority**: This is the defaulting behavior the feature exists to provide.

**Independent Test**: Submit a Cluster with `NutanixMetro/` control-plane failure domains and a Nutanix CSI `volume` StorageClassConfig that has no `computeAffinity` parameter. After CREATE, the Cluster variable contains `computeAffinity: DISABLED` on that StorageClassConfig.

**Acceptance Scenarios**:

1. **Given** a metro Cluster (control-plane or worker failure domain prefixed `NutanixMetro/` or `NutanixMetroSite/`) with one or more Nutanix CSI StorageClassConfigs that omit `computeAffinity`, **When** the Cluster is created or updated, **Then** each of those StorageClassConfigs has `parameters.computeAffinity` set to `DISABLED`.
2. **Given** a metro Cluster whose StorageClassConfig already has `computeAffinity: DISABLED`, **When** the Cluster is admitted, **Then** the value is left unchanged and admission succeeds.
3. **Given** a non-metro Cluster with Nutanix CSI StorageClassConfigs that omit `computeAffinity`, **When** the Cluster is admitted, **Then** `computeAffinity` is not added.

### User Story 2 - Explicit non-DISABLED computeAffinity is rejected on metro (Priority: P1)

An operator sets `computeAffinity` to any value other than `DISABLED` on a metro Cluster. Admission must fail with a message that the value is not supported for a metro cluster.

**Why this priority**: Prevents an invalid StorageClass from being applied to a metro workload cluster.

**Independent Test**: Submit a metro Cluster with `parameters.computeAffinity: ENABLED` (or any other non-`DISABLED` value). Admission is denied and the response message includes that it is not supported for the metro cluster.

**Acceptance Scenarios**:

1. **Given** a metro Cluster with Nutanix CSI StorageClassConfig `computeAffinity` set to a value other than `DISABLED`, **When** the Cluster is created or updated, **Then** admission is denied and the message states that the value is not supported for the metro cluster.
2. **Given** a non-metro Cluster with `computeAffinity` set to a non-`DISABLED` value, **When** the Cluster is admitted, **Then** admission succeeds (this webhook does not constrain non-metro clusters).

### User Story 3 - Upgrade is a no-op for topology mutation patches (Priority: P1)

Operators upgrading CAREN must not see Machine rollouts from this change.

**Why this priority**: Constitution principle VII.

**Independent Test**: Review the diff; no files under `pkg/handlers/*/mutation/` change. Admission mutates Cluster topology *variables* (CSI StorageClass parameters only), which does not change mutation-handler `GeneratePatches` output.

**Acceptance Scenarios**:

1. **Given** existing managed Clusters, **When** CAREN is upgraded to include this webhook, **Then** no topology mutation handler names or patch outputs change.
2. **Given** a non-metro Cluster, **When** CAREN is upgraded, **Then** Cluster admission does not add `computeAffinity`.

### Edge Cases

- Cluster has no topology: skip (existing cluster webhook matchConditions already require topology).
- Cluster has topology but no `clusterConfig` variable, no CSI addon, or no `nutanix` CSI provider: skip defaulting and validation.
- Multiple StorageClassConfigs: default or reject each independently; one invalid class denies the whole request.
- Metro detected only from worker MachineDeployment `failureDomain` (control plane not metro-prefixed): still treated as metro, consistent with CSI MPIO detection.
- Empty `computeAffinity` (`""`): treated as unset and defaulted to `DISABLED` on metro (not as an explicit unsupported value).
- DELETE operations: skip.

## Requirements

### Functional Requirements

- **FR-001**: Mutating Cluster admission MUST set `addons.csi.providers.nutanix.storageClassConfigs.<name>.parameters.computeAffinity` to `DISABLED` for every Nutanix CSI StorageClassConfig on a metro Cluster when the key is absent or empty.
- **FR-002**: Validating Cluster admission MUST deny a metro Cluster when any Nutanix CSI StorageClassConfig has a non-empty `computeAffinity` other than `DISABLED`, with an error stating it is not supported for the metro cluster.
- **FR-003**: A Cluster is metro when any control-plane Nutanix failure domain or any topology worker MachineDeployment failure domain has prefix `NutanixMetro/` or `NutanixMetroSite/`.
- **FR-004**: Non-metro Clusters MUST NOT be defaulted or rejected by this logic.
- **FR-005**: Mutating webhook MUST NOT overwrite a user-provided non-empty `computeAffinity` (validation rejects unsupported values instead).
- **FR-006**: Logic MUST run on Cluster CREATE and UPDATE, matching existing `/mutate-cluster` and `/validate-cluster` webhook configuration.

### Key Entities

- **Metro Cluster**: CAPI Cluster whose topology failure domains reference NutanixMetro or NutanixMetroSite.
- **Nutanix CSI StorageClassConfig**: `clusterConfig.addons.csi.providers.nutanix.storageClassConfigs` entries; parameters are passed through to workload-cluster StorageClass objects.

## Success Criteria

- **SC-001**: Metro Cluster CREATE without `computeAffinity` results in `DISABLED` on every Nutanix CSI StorageClassConfig.
- **SC-002**: Metro Cluster CREATE/UPDATE with `computeAffinity` other than `DISABLED` is denied with a metro-not-supported message.
- **SC-003**: Unit tests cover defaulting, rejection, non-metro no-op, already-DISABLED, and missing CSI configs.
- **SC-004**: No topology mutation handler version bump.
