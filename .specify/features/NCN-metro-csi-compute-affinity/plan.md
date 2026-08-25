<!--
 Copyright 2026 Nutanix. All rights reserved.
 SPDX-License-Identifier: Apache-2.0
-->

# Implementation Plan: Default and validate CSI computeAffinity for metro clusters

**Branch**: `cursor/metro-csi-compute-affinity-43d1`
**Date**: 2026-08-25
**Spec**: [./spec.md](./spec.md)

## Summary

Add Cluster admission defaulting and validation so metro Clusters always get Nutanix CSI StorageClass parameter `computeAffinity: DISABLED`, and reject any other explicit value. Reuse the existing metro failure-domain prefix detection currently private to the Nutanix CSI lifecycle handler.

## Technical Context

**Language/Version**: Go (see `go.mod`)
**Primary Dependencies**: controller-runtime admission webhooks, CAPI Cluster topology variables
**Testing**: testify unit tests
**Target Platform**: management-cluster admission webhooks
**Project Type**: single Go module
**Constraints**: No mutation-handler version bump; do not overwrite explicit non-empty `computeAffinity`
**Scale/Scope**: helpers + cluster webhook package + CSI handler call-site + docs

## Constitution Check

| Principle | Status | Notes |
| --- | --- | --- |
| I. API-First | Pass | No API type changes; uses existing StorageClassConfig.Parameters map |
| II. Handler-per-Provider | Pass | Nutanix CSI and Nutanix metro only |
| III. Library-First | Pass | Shared metro detection in `pkg/helpers`; webhook and CSI handler consume it |
| IV. Tests Required | Pass | Failing unit tests first |
| V. Code Style | Pass | Existing import aliases |
| VI. Dependency Management | Pass | No new deps |
| VII. Handler Version Safety | Pass | No `pkg/handlers/*/mutation/` edits |
| VIII. Handler Documentation | Pass | User-facing note under Nutanix customization docs |

No violations.

## Project Structure

```text
.specify/features/NCN-metro-csi-compute-affinity/
├── spec.md
└── plan.md

pkg/helpers/
├── metro.go
└── metro_test.go

pkg/webhook/cluster/
├── defaulter.go
├── validator.go
├── metro_storageclass.go
└── metro_storageclass_test.go

pkg/handlers/lifecycle/csi/nutanix/handler.go

docs/content/customization/nutanix/metro-csi.md
```

## Tasks

1. Extract `IsMetroCluster` (and metro FD prefixes) to `pkg/helpers`; point CSI handler at it. Tests for control-plane FD, worker FD, and non-metro.
2. Write failing tests for mutating defaulting and validating rejection.
3. Implement mutating defaulter and validating webhook; register on existing `/mutate-cluster` and `/validate-cluster`.
4. Document metro CSI `computeAffinity` behavior.
5. Run unit tests for `pkg/helpers` and `pkg/webhook/cluster`.
