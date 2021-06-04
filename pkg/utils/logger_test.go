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

// Package utils ...
package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestGetContextLogger(t *testing.T) {
	ctxLog, reqID := GetContextLogger(context.Background(), true)
	assert.NotNil(t, ctxLog)
	assert.NotNil(t, reqID)

	ctxLog, reqID = GetContextLogger(context.Background(), false)
	assert.NotNil(t, ctxLog)
	assert.NotNil(t, reqID)
}

func TestGetContextLoggerWithRequestID(t *testing.T) {
	requestIDIn := "1000"
	ctxLog, reqID := GetContextLoggerWithRequestID(context.Background(), true, &requestIDIn)
	assert.NotNil(t, ctxLog)
	assert.NotNil(t, requestIDIn)

	ctxLog, reqID = GetContextLoggerWithRequestID(context.Background(), true, nil)
	assert.NotNil(t, ctxLog)
	assert.NotNil(t, reqID)
}
