/**
 * Copyright 2021 IBM Corp.
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

// Package messages ...
package messages

import (
	"testing"

	"github.com/IBM/ibm-csi-common/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetCSIError(t *testing.T) {
	MessagesEn = InitMessages()
	ctxLog, reqID := utils.GetContextLogger(context.Background(), false)

	err := GetCSIError(ctxLog, MethodUnimplemented, reqID, nil)
	assert.NotNil(t, err)

	err1 := GetCSIError(ctxLog, MethodUnimplemented, reqID, err)
	assert.NotNil(t, err1)
}

func TestGetCSIMessage(t *testing.T) {
	message := GetCSIMessage(MethodUnimplemented, "create")
	assert.NotNil(t, message)
}

func TestGetCSIBackendErrorKeepsExistingStatuses(t *testing.T) {
	MessagesEn = InitMessages()
	logger := zap.NewNop()
	forbidden := Message{CSIError: "RC:403 Forbidden"}
	unavailable := Message{CSIError: "RC:503 Service Unavailable"}

	assert.Equal(t, codes.InvalidArgument, status.Code(GetCSIBackendError(logger, "request-id", forbidden)))
	assert.Equal(t, codes.Internal, status.Code(GetCSIBackendError(logger, "request-id", unavailable)))
}
