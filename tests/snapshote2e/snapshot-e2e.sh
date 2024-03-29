#!/bin/bash

#/******************************************************************************
# Copyright 2022 IBM Corp.
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

# Pre requisites
# 1. Golang must be installed and GOPATH must be set
# 2. KUBECONFIG path must be set to a vpc gen2 cluster
# 3. git must be installed and be configured with the ssh key from where this script is running.
# 4. ibmcloud command must be installed with 'ks' utility and should be logged into via CLI
# 5. ginkgo cli needs to be installed - this is taken care of in the script as of now.

# Run the script - bash snapshot-e2e.sh
# error() - prints the error message passed to it, and exists from the script

EXPECTED_VERSION=

error() {
     if [[ $? != 0 ]]; then
         echo "$1"
         exit 1
     fi
}

# Print KUBECONFIG
echo "KUBECONFIG is set to $KUBECONFIG"

#Get cluster name
IN=$(kubectl config current-context)
error "Unable to fetch kubectl config"
IFS='/' read -ra CLUSTER_NAME <<< "$IN"
echo "Cluster Name - $CLUSTER_NAME"

# Fetching pods in kube-system namespace to make sure, snapshot controller, and driver with sidecar is deployed.
echo "Fetching pods in kube-system namespace"
kubectl get pods -n kube-system

# Checking if controller pod is up
echo "Checking if controller pod is up"
kubectl get pods -n kube-system | grep ibm-vpc-block-csi-controller
error "Driver is not enabled"

# Sync ibm-csi-common
mkdir -p "$GOPATH/src" "$GOPATH/bin" && sudo chmod -R 777 "$GOPATH"
error "Unable to create src under GOPATH"
mkdir -p $GOPATH/src/ibm-csi-common
rsync -az ./ibm-csi-common $GOPATH/src/
cd $GOPATH/src/ibm-csi-common
DIR="$(pwd)"
echo "Present working directory: $DIR"

echo "Installing ginkgo"
export PATH=$PATH:$GOPATH/bin
export CGO_ENABLED=0
go install github.com/onsi/ginkgo/ginkgo@v1.16.4
error "Error installing ginkgo"

echo "Starting snapshot basic e2e tests for vpc block storage"
touch snapshote2e_test_result.log
export E2E_TEST_RESULT=$GOPATH/src/ibm-csi-common/snapshote2e_test_result.log
ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[snapshot\]"  ./tests/snapshote2e
cat $E2E_TEST_RESULT
rm -rf $GOPATH/src/ibm-csi-common
export CGO_ENABLED=1
echo "Finished snapshot basic e2e tests for vpc block storage"






