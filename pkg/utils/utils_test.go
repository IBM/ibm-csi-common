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

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
)

func TestRoundUpBytes(t *testing.T) {
	var sizeInBytes int64 = 1024
	actual := RoundUpBytes(sizeInBytes)
	if actual != 1*GiB {
		t.Fatalf("Wrong result for RoundUpBytes. Got: %d", actual)
	}
}

func TestBytesToGiB(t *testing.T) {
	var sizeInBytes int64 = 5 * GiB

	actual := BytesToGiB(sizeInBytes)
	if actual != 5 {
		t.Fatalf("Wrong result for BytesToGiB. Got: %d", actual)
	}
}

func TestListContainsSubstr(t *testing.T) {
	testCases := []struct {
		testCaseName   string
		inputMainStr   []string
		inputSubStr    string
		expectedOutput bool
	}{
		{
			testCaseName:   "Valid sub-string",
			inputMainStr:   []string{"str1", "str2", "str3"},
			inputSubStr:    "str3",
			expectedOutput: true,
		},
		{
			testCaseName:   "Empty sub-string",
			inputMainStr:   []string{"str1", "str2", "str3"},
			inputSubStr:    "",
			expectedOutput: false,
		},
		{
			testCaseName:   "Sub-string not exists",
			inputMainStr:   []string{"str1", "str2", "str3"},
			inputSubStr:    "str4",
			expectedOutput: false,
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.testCaseName, func(t *testing.T) {
			actualStatus := ListContainsSubstr(testcase.inputMainStr, testcase.inputSubStr)
			assert.Equal(t, testcase.expectedOutput, actualStatus)
		})
	}
}

func TestNewVolumeCapabilityAccessMode(t *testing.T) {
	testCases := []struct {
		testCaseName   string
		inputMode      csi.VolumeCapability_AccessMode_Mode
		expectedOutput *csi.VolumeCapability_AccessMode
	}{
		{
			testCaseName:   "Single writer node acess mode",
			inputMode:      csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			expectedOutput: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER},
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.testCaseName, func(t *testing.T) {
			volCap := NewVolumeCapabilityAccessMode(testcase.inputMode)
			assert.Equal(t, testcase.expectedOutput.Mode, volCap.Mode)
		})
	}
}

func TestNewControllerServiceCapability(t *testing.T) {
	testCases := []struct {
		testCaseName   string
		inputCap       csi.ControllerServiceCapability_RPC_Type
		expectedOutput *csi.ControllerServiceCapability
	}{
		{
			testCaseName: "Valid capability",
			inputCap:     csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
			expectedOutput: &csi.ControllerServiceCapability{Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
				},
			},
			},
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.testCaseName, func(t *testing.T) {
			volCap := NewControllerServiceCapability(testcase.inputCap)
			assert.Equal(t, testcase.expectedOutput.GetRpc(), volCap.GetRpc())
		})
	}
}

func TestNewNodeServiceCapability(t *testing.T) {
	testCases := []struct {
		testCaseName   string
		inputNodeCap   csi.NodeServiceCapability_RPC_Type
		expectedOutput *csi.NodeServiceCapability
	}{
		{
			testCaseName: "Valid node service capability",
			inputNodeCap: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
			expectedOutput: &csi.NodeServiceCapability{Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
				},
			},
			},
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.testCaseName, func(t *testing.T) {
			nodeCap := NewNodeServiceCapability(testcase.inputNodeCap)
			assert.Equal(t, testcase.expectedOutput.GetRpc(), nodeCap.GetRpc())
		})
	}
}

func TestGetEnv(t *testing.T) {
	envKey := "NO_ENV_VARIABLE"
	envValue := getEnv(envKey)
	assert.Equal(t, "", envValue)
}
