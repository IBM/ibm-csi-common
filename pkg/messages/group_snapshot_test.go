/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package messages

import (
	"testing"

	providerError "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetCSIGroupSnapshotBackendError(t *testing.T) {
	testCases := []struct {
		name      string
		operation GroupSnapshotOperation
		err       error
		expected  codes.Code
	}{
		{
			name:      "duplicate name during create keeps creation pending",
			operation: GroupSnapshotCreate,
			err: providerError.Message{
				Code:         "FailedToCreateGroupSnapshot",
				BackendError: "Trace Code:create-trace, Code:snapshot_consistency_groups_name_duplicate, RC:409 Conflict",
			},
			expected: codes.Aborted,
		},
		{
			name:      "invalid resource group during create",
			operation: GroupSnapshotCreate,
			err: providerError.Message{
				Code:         "FailedToCreateGroupSnapshot",
				BackendError: "Trace Code:create-trace, Code:snapshot_consistency_groups_resource_group_invalid, RC:400 Bad Request",
			},
			expected: codes.InvalidArgument,
		},
		{
			name:      "invalid resource group even without the HTTP status",
			operation: GroupSnapshotCreate,
			err: providerError.Message{
				Code:         "FailedToCreateGroupSnapshot",
				BackendError: "Trace Code:create-trace, Code:snapshot_consistency_groups_resource_group_invalid, Description:Invalid resource group",
				RC:           500,
			},
			expected: codes.InvalidArgument,
		},
		{
			name: "provider member source-volume details are incomplete",
			err: providerError.Message{
				Code: "GroupSnapshotMemberDetailsUnavailable",
				Type: providerError.RetrivalFailed,
				RC:   409,
			},
			expected: codes.Aborted,
		},
		{
			name:      "create member lookup failed after the group may have been created",
			operation: GroupSnapshotCreate,
			err: providerError.Message{
				Code:         "GroupSnapshotMemberLookupFailed",
				BackendError: "Trace Code:member-lookup-trace, Code:snapshots_not_authorized, RC:403 Forbidden",
			},
			expected: codes.Aborted,
		},
		{
			name:      "create received an invalid group response",
			operation: GroupSnapshotCreate,
			err:       providerError.Message{Code: "InvalidGroupSnapshotResponse", RC: 500},
			expected:  codes.Aborted,
		},
		{
			name: "member lookup permission denied",
			err: providerError.Message{
				Code:         "GroupSnapshotMemberLookupFailed",
				Type:         providerError.RetrivalFailed,
				RC:           500,
				BackendError: "Trace Code:member-lookup-trace, Code:snapshots_not_authorized, RC:403 Forbidden",
			},
			expected: codes.PermissionDenied,
		},
		{
			name: "member lookup authentication failed",
			err: providerError.Message{
				Code:         "GroupSnapshotMemberLookupFailed",
				Type:         providerError.RetrivalFailed,
				RC:           500,
				BackendError: "Trace Code:member-lookup-trace, Code:token_invalid, RC:401 Unauthorized",
			},
			expected: codes.Unauthenticated,
		},
		{
			name: "member lookup rate limited",
			err: providerError.Message{
				Code:         "GroupSnapshotMemberLookupFailed",
				Type:         providerError.RetrivalFailed,
				RC:           500,
				BackendError: "Trace Code:member-lookup-trace, Code:snapshots_too_many_requests, RC:429 Too Many Requests",
			},
			expected: codes.ResourceExhausted,
		},
		{
			name: "member lookup internal backend error",
			err: providerError.Message{
				Code:         "GroupSnapshotMemberLookupFailed",
				Type:         providerError.RetrivalFailed,
				RC:           500,
				BackendError: "Trace Code:member-lookup-trace, Code:internal_error, RC:500 Internal Server Error",
			},
			expected: codes.Unavailable,
		},
		{
			name: "source volume is not attached",
			err: providerError.Message{
				Type:         providerError.ProvisioningFailed,
				BackendError: "Code:snapshots_source_volume_not_attached, RC:409",
			},
			expected: codes.FailedPrecondition,
		},
		{
			name: "source volume not found",
			err: providerError.Message{
				Type:         providerError.ProvisioningFailed,
				BackendError: "Code:snapshots_source_volume_not_found, RC:404",
			},
			expected: codes.NotFound,
		},
		{
			name: "source volume busy",
			err: providerError.Message{
				Type:         providerError.ProvisioningFailed,
				BackendError: "Code:snapshots_source_volume_busy, RC:409",
			},
			expected: codes.Aborted,
		},
		{
			name: "snapshot service unavailable",
			err: providerError.Message{
				Type:         providerError.ProvisioningFailed,
				BackendError: "Code:snapshots_service_unavailable, RC:503",
			},
			expected: codes.Unavailable,
		},
		{
			name: "generic backend conflict",
			err: providerError.Message{
				Type:         providerError.DeletionFailed,
				BackendError: "Code:invalid_state, RC:409",
			},
			expected: codes.FailedPrecondition,
		},
		{
			name: "backend capacity exhausted",
			err: providerError.Message{
				Type:         providerError.ProvisioningFailed,
				BackendError: "Code:limit_reached, RC:429",
			},
			expected: codes.ResourceExhausted,
		},
		{
			name:     "invalid backend request",
			err:      providerError.Message{Type: providerError.InvalidRequest},
			expected: codes.InvalidArgument,
		},
		{
			name:      "delete group not found",
			operation: GroupSnapshotDelete,
			err:       providerError.Message{BackendError: "Code:snapshot_consistency_groups_not_found, RC:404"},
			expected:  codes.NotFound,
		},
		{
			name:     "permission denied",
			err:      providerError.Message{Type: providerError.PermissionDenied},
			expected: codes.PermissionDenied,
		},
		{
			name:     "unknown provider error",
			err:      providerError.Message{Type: providerError.ProvisioningFailed},
			expected: codes.Internal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			operation := tc.operation
			if operation == "" {
				operation = GroupSnapshotGet
			}
			err := GetCSIGroupSnapshotBackendError(zap.NewNop(), "request-id", operation, "group context", tc.err)
			assert.Equal(t, tc.expected, status.Code(err))
			message := status.Convert(err).Message()
			assert.Contains(t, message, "RequestID: request-id")
			assert.Contains(t, message, "BackendError: group context: "+tc.err.Error())
		})
	}
}

func TestGetCSIGroupSnapshotBackendErrorNil(t *testing.T) {
	assert.NoError(t, GetCSIGroupSnapshotBackendError(zap.NewNop(), "request-id", GroupSnapshotGet, "group context", nil))
}
