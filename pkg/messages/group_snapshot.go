/*
 * Copyright 2026 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package messages

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	providerError "github.com/IBM/ibmcloud-volume-interface/lib/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	msg := Message{
		Type:         groupSnapshotBackendCode(operation, err),
		RequestID:    requestID,
		BackendError: fmt.Sprintf("%s: %v", description, err),
		Action:       "Please check 'BackendError' tag for more details",
	}
	logger.Error("CSI request failed", zap.Error(msg))
	return status.Error(msg.Type, msg.Info())
}

// A VPC failure reaches us as text inside the provider error, for example:
//
//	Trace Code:<id>, Code:snapshots_source_volume_busy, Description:<text>, RC:409 Conflict
//
// These patterns read the VPC error code and the HTTP status from that text.
var (
	vpcCodeRe   = regexp.MustCompile(`(?:^|, )Code:([a-z0-9_]+)`)
	vpcStatusRe = regexp.MustCompile(`, RC:(\d{3})`)
)

// vpcCodeToCSICode maps a VPC error code to the CSI status returned to the snapshotter.
var vpcCodeToCSICode = map[string]codes.Code{
	"snapshot_consistency_groups_resource_group_invalid": codes.InvalidArgument,
	"snapshots_source_volume_not_found":                  codes.NotFound,
	"snapshot_consistency_groups_not_found":              codes.NotFound,
	"snapshots_not_found":                                codes.NotFound,
	"snapshots_source_volume_not_attached":               codes.FailedPrecondition,
	"snapshots_source_volume_busy":                       codes.Aborted,
	"snapshots_service_unavailable":                      codes.Unavailable,
}

// httpStatusToCSICode maps an HTTP status to a CSI status. Any 5xx is handled separately.
var httpStatusToCSICode = map[int]codes.Code{
	400: codes.InvalidArgument,
	401: codes.Unauthenticated,
	403: codes.PermissionDenied,
	404: codes.NotFound,
	409: codes.FailedPrecondition, // A 409 doesn't always mean a duplicate name.
	429: codes.ResourceExhausted,
}

// providerTypeToCSICode maps the provider's error category to a CSI status.
var providerTypeToCSICode = map[string]codes.Code{
	providerError.EntityNotFound:    codes.NotFound,
	providerError.InvalidRequest:    codes.InvalidArgument,
	providerError.PermissionDenied:  codes.PermissionDenied,
	providerError.Unauthenticated:   codes.Unauthenticated,
	providerError.FailedAccessToken: codes.Unauthenticated,
}

// groupSnapshotBackendCode picks the CSI status for a VGS backend error.
func groupSnapshotBackendCode(operation GroupSnapshotOperation, err error) codes.Code {
	var providerMsg providerError.Message
	errors.As(err, &providerMsg)
	vpcCode, httpStatus := readVPCError(providerMsg)

	// 1. The group may already exist in VPC. Aborted keeps the request pending,
	//    and the retry finds the group by name instead of creating another one.
	if groupMayExist(operation, providerMsg.Code, vpcCode) {
		return codes.Aborted
	}
	// 2. A VPC error code we know.
	if code, ok := vpcCodeToCSICode[vpcCode]; ok {
		return code
	}
	// 3. The HTTP status. A 5xx may still be running in VPC, so report it as temporary.
	if code, ok := httpStatusToCSICode[httpStatus]; ok {
		return code
	}
	if httpStatus >= 500 {
		return codes.Unavailable
	}
	// 4. The provider's own error category.
	if code, ok := providerTypeToCSICode[providerMsg.Type]; ok {
		return code
	}
	return codes.Internal
}

// readVPCError returns the VPC error code and HTTP status carried by a provider error.
// When VPC sent no status (for example a network timeout), the provider's own status is used.
func readVPCError(providerMsg providerError.Message) (vpcCode string, httpStatus int) {
	if m := vpcCodeRe.FindStringSubmatch(providerMsg.BackendError); m != nil {
		vpcCode = m[1]
	}
	httpStatus = providerMsg.RC
	if m := vpcStatusRe.FindStringSubmatch(providerMsg.BackendError); m != nil {
		httpStatus, _ = strconv.Atoi(m[1])
	}
	return vpcCode, httpStatus
}

// groupMayExist reports whether VPC may already hold the group even though the call failed.
func groupMayExist(operation GroupSnapshotOperation, providerCode, vpcCode string) bool {
	// VPC returned the group, but its member details are not complete yet.
	if providerCode == "GroupSnapshotMemberDetailsUnavailable" {
		return true
	}
	if operation != GroupSnapshotCreate {
		return false
	}

	switch providerCode {
	case "GroupSnapshotMemberLookupFailed", "InvalidGroupSnapshotResponse":
		// The group was created, but the driver can't describe it yet.
		return true
	}

	// An earlier attempt may have created the group and lost the response.
	return vpcCode == "snapshot_consistency_groups_name_duplicate"
}
