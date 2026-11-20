/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package messages

import (
	"fmt"
	"strings"

	providerError "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

// GroupSnapshotOperation identifies the CSI request that hit the backend error.
type GroupSnapshotOperation string

const (
	GroupSnapshotCreate GroupSnapshotOperation = "create"
	GroupSnapshotGet    GroupSnapshotOperation = "get"
	GroupSnapshotDelete GroupSnapshotOperation = "delete"
)

// GetCSIGroupSnapshotBackendError reports a VGS backend failure with its CSI status.
func GetCSIGroupSnapshotBackendError(logger *zap.Logger, requestID string, operation GroupSnapshotOperation, description string, err error) error {
	if err == nil {
		return nil
	}
	return returnBackendError(logger, Message{
		Type:         groupSnapshotBackendCode(operation, err),
		RequestID:    requestID,
		BackendError: fmt.Sprintf("%s: %v", description, err),
		Action:       "Please check 'BackendError' tag for more details",
	})
}

func groupSnapshotBackendCode(operation GroupSnapshotOperation, err error) codes.Code {
	providerMsg, _ := err.(providerError.Message)
	switch providerMsg.Code {
	case "GroupSnapshotMemberDetailsUnavailable":
		// Aborted keeps Create pending while member details are missing.
		return codes.Aborted
	case "GroupSnapshotMemberLookupFailed", "InvalidGroupSnapshotResponse":
		if operation == GroupSnapshotCreate {
			// The group may already exist, but the driver can't return its ID yet.
			// Aborted tells the snapshotter to keep Create pending.
			return codes.Aborted
		}
	}

	// The provider passes along the VPC code and HTTP status as text.
	errorText := strings.ToLower(strings.ReplaceAll(err.Error(), " ", ""))
	switch {
	case operation == GroupSnapshotCreate && strings.Contains(errorText, "code:snapshot_consistency_groups_name_duplicate,"):
		// VPC may have created the group. Aborted keeps Create pending.
		// When the snapshotter retries, the driver checks for the group by name.
		return codes.Aborted
	case strings.Contains(errorText, "code:snapshot_consistency_groups_resource_group_invalid,"):
		return codes.InvalidArgument
	case strings.Contains(errorText, "snapshots_source_volume_not_found"),
		strings.Contains(errorText, "snapshot_consistency_groups_not_found"),
		strings.Contains(errorText, "snapshots_not_found"):
		// VPC couldn't find the group, a member snapshot, or a source volume
		return codes.NotFound
	case strings.Contains(errorText, "snapshots_source_volume_not_attached"):
		// Attach the source volume before trying again
		return codes.FailedPrecondition
	case strings.Contains(errorText, "snapshots_source_volume_busy"):
		// A busy volume may free up. Aborted keeps Create pending.
		return codes.Aborted
	case strings.Contains(errorText, "snapshots_service_unavailable"):
		// Try the same request when the snapshot service is back
		return codes.Unavailable
	}
	switch {
	case strings.Contains(errorText, "rc:400"):
		return codes.InvalidArgument
	case strings.Contains(errorText, "rc:401"):
		return codes.Unauthenticated
	case strings.Contains(errorText, "rc:403"):
		return codes.PermissionDenied
	case strings.Contains(errorText, "rc:404"):
		return codes.NotFound
	case strings.Contains(errorText, "rc:409"):
		// Something in VPC is blocking the request. A 409 doesn't always mean a duplicate name
		return codes.FailedPrecondition
	case strings.Contains(errorText, "rc:429"):
		// Too many requests. Slow down before trying again
		return codes.ResourceExhausted
	case strings.Contains(errorText, "rc:5"):
		// VPC hit a server error. Keep Create open because it may still be running
		return codes.Unavailable
	}

	// If the VPC error doesn't match, use the provider's error category.
	switch providerMsg.Type {
	case providerError.EntityNotFound:
		// The provider couldn't find the requested data
		return codes.NotFound
	case providerError.InvalidRequest:
		// The provider rejected the input. Fix the request first
		return codes.InvalidArgument
	case providerError.PermissionDenied:
		return codes.PermissionDenied
	case providerError.Unauthenticated, providerError.FailedAccessToken:
		return codes.Unauthenticated
	default:
		return codes.Internal
	}
}
