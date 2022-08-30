#!/bin/bash

#/******************************************************************************
# Copyright 2021 IBM Corp.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# *****************************************************************************/

GOPATH=$GOPATH
VPC_BLOCK_CSI_HOME="$GOPATH/src/github.com/IBM/ibm-csi-common"

# E2E Execution
go clean -modcache
export GO111MODULE=on
go get -u github.com/onsi/ginkgo/ginkgo

set +e
ginkgo -v -nodes=1 --focus="\[pvg-e2e\] \[snapshot\]" ./tests/e2e | tee -a block-vpc-csi-ginkgo-log.txt
rc=$?
echo "Finished armada storage basic e2e tests"
exit 0
