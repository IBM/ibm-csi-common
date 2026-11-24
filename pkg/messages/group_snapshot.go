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
	"fmt"
	"regexp"
	"strconv"
	"strings"

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

var (
	// The provider turns a VPC error into text such as "Trace Code:<id>, Code:<vpc_code>, Description:<text>, RC:404 Not Found".
	// These patterns pull out only the VPC "Code:" and "RC:" values
	vpcCodeRe = regexp.MustCompile(`\bcode:([a-z0-9_]+)`)
	vpcRCRe   = regexp.MustCompile(`\brc:(\d{3})`)
)

// vpcCodeToCSICode maps a specific VPC error code to the CSI status code returned to the snapshotter.
var vpcCodeToCSICode = map[string]codes.Code{
	"snapshot_consistency_groups_resource_group_invalid": codes.InvalidArgument,
	"snapshots_source_volume_not_found":                  codes.NotFound,           // A source volume in the request doesn't exist in VPC, so the request can't succeed as written.
	"snapshot_consistency_groups_not_found":              codes.NotFound,           // The snapshot consistency group doesn't exist in VPC (for example, it was already deleted).
	"snapshots_not_found":                                codes.NotFound,           // A member snapshot of the group doesn't exist in VPC.
	"snapshots_source_volume_not_attached":               codes.FailedPrecondition, // A source volume isn't attached to an instance.
	"snapshots_source_volume_busy":                       codes.Aborted,            // A source volume is busy right now. Keep Create pending and let the snapshotter retry.
	"snapshots_service_unavailable":                      codes.Unavailable,        // The VPC snapshot service is temporarily down.
}

// groupSnapshotBackendCode picks the CSI status code for a VGS backend error.
// It checks, in order: the provider's own error code, the VPC error code, the HTTP status (RC), and finally the provider's error category.
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

	// The provider passes the VPC code and HTTP status along as text.
	// Lowercase it and drop spaces so the patterns above match reliably.
	errorText := strings.ToLower(strings.ReplaceAll(err.Error(), " ", ""))

	// The text can hold more than one "Code:" (the trace code, then the VPC code), extract VPC code.
	for _, m := range vpcCodeRe.FindAllStringSubmatch(errorText, -1) {
		vpcCode := m[1]
		if operation == GroupSnapshotCreate && vpcCode == "snapshot_consistency_groups_name_duplicate" {
			// A group with this name already exists. It may have been created by an earlier
			// attempt whose response was lost. Aborted keeps Create pending, and on the retry
			// the driver looks the group up by name instead of creating another one.
			return codes.Aborted
		}
		if code, ok := vpcCodeToCSICode[vpcCode]; ok {
			return code
		}
	}

	// No known VPC code matched, so fall back to the HTTP status VPC returned.
	if m := vpcRCRe.FindStringSubmatch(errorText); m != nil {
		rc, _ := strconv.Atoi(m[1])
		switch {
		case rc == 400:
			return codes.InvalidArgument // VPC rejected the request as malformed.
		case rc == 401:
			return codes.Unauthenticated
		case rc == 403:
			return codes.PermissionDenied
		case rc == 404:
			return codes.NotFound
		case rc == 409:
			// VPC refused the request because of the current state of a resource. A 409 doesn't always mean a duplicate name.
			return codes.FailedPrecondition
		case rc == 429:
			// VPC is rate limiting us. Back off and retry later.
			return codes.ResourceExhausted
		case rc >= 500:
			// VPC had a server-side error. The request may still be running, so report it as temporary and let the snapshotter retry.
			return codes.Unavailable
		}
	}

	// Neither a VPC code nor an HTTP status matched, so use the provider's own error category.
	switch providerMsg.Type {
	case providerError.EntityNotFound:
		return codes.NotFound // The provider couldn't find the requested data.
	case providerError.InvalidRequest:
		return codes.InvalidArgument // The provider rejected the input. The request has to be fixed first.
	case providerError.PermissionDenied:
		return codes.PermissionDenied
	case providerError.Unauthenticated, providerError.FailedAccessToken:
		return codes.Unauthenticated
	default:
		return codes.Internal // Anything we don't recognize is treated as an internal error.
	}
}
