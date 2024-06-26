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

// Package metadata ...
package metadata

import (
	"testing"

	"github.com/IBM/ibm-csi-common/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestNewNodeMetadata(t *testing.T) {
	// Creating test logger
	logger, teardown := utils.GetTestLogger(t)
	defer teardown()
	nodeInfo := NodeInfoManager{NodeName: "mynode"}
	nodeMeta, err := nodeInfo.NewNodeMetadata(logger)

	// Error will be there as there is no kubernetes running
	assert.NotNil(t, err)
	assert.Nil(t, nodeMeta)

	// statically creating
	nodeMetadata := &nodeMetadataManager{zone: "myzone", region: "myregion", workerID: "myworkerid"}
	assert.Equal(t, "myzone", nodeMetadata.GetZone())
	assert.Equal(t, "myregion", nodeMetadata.GetRegion())
	assert.Equal(t, "myworkerid", nodeMetadata.GetWorkerID())
}

func TestGetZone(t *testing.T) {
	fakeNodeData := FakeNodeMetadata{}
	fakeNodeData.GetRegionReturns("testregion")
	fakeNodeData.GetZoneReturns("testzone")
	fakeNodeData.GetWorkerIDReturns("testworkerid")

	assert.Equal(t, "testzone", fakeNodeData.GetZone())
}

func TestGetRegion(t *testing.T) {
	fakeNodeData := FakeNodeMetadata{}
	fakeNodeData.GetRegionReturns("testregion")
	fakeNodeData.GetZoneReturns("testzone")
	fakeNodeData.GetWorkerIDReturns("testworkerid")

	assert.Equal(t, "testregion", fakeNodeData.GetRegion())
}

func TestGetWorkerID(t *testing.T) {
	fakeNodeData := FakeNodeMetadata{}
	fakeNodeData.GetRegionReturns("testregion")
	fakeNodeData.GetZoneReturns("testzone")
	fakeNodeData.GetWorkerIDReturns("testworkerid")

	assert.Equal(t, "testworkerid", fakeNodeData.GetWorkerID())
}
