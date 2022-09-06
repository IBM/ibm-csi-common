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

# Pre requisites
# 1. Golang must be installed and GOPATH must be set
# 2. KUBECONFIG path must be set to a vpc gen2 cluster
# 3. git must be installed and be configured with the ssh key from where this script is running.
# 4. ibmcloud command must be installed with 'ks' utility and should be logged into via CLI
# 5. ginkgo cli needs to be installed 

# Run the script - bash snapshot-e2e.sh
# error() - prints the error message passed to it, and exists from the script
error() {
     if [[ $? != 0 ]]; then
         echo "$1"
         exit
     fi
}

# enableAddon() - disables the existing vpc-block-csi-driver addon and enables the 5.0 addon from which snapshot is supported.
enableAddon() {
    ibmcloud ks cluster addon disable  vpc-block-csi-driver -f -c $1
    # waiting for addon to be disabled
    sleep 30s

    # Enable 5.0 vpc-block-csi-driver addon
    # Fetching addon version
    addonVersion=$(ibmcloud ks cluster addon versions | grep vpc-block-csi-driver | grep 5.0 | awk '{print $2}')
    error "Unable to fetch vpc-block-csi-driver addon version"

    # Enabling addon
    ibmcloud ks cluster addon enable vpc-block-csi-driver   -c $1 --version $addonVersion
    sleep 20s
}

# Print KUBECONFIG
echo $KUBECONFIG

#Get cluster name
IN=$(kubectl config current-context)
error "Unable to fetch kubectl config"
IFS='/' read -ra CLUSTER_NAME <<< "$IN"
echo "Cluster Name - $CLUSTER_NAME"

#Fetch cluster addon version
addon_version=$(ibmcloud ks cluster addon ls -c $CLUSTER_NAME | grep vpc-block-csi-driver | awk '{print $2}')
error "Unable to fetch vpc-block-csi-driver addon version"
echo "$addon_version"

#Check if the addon version is 5.0, if it is not disable the existing one, and enable 5.0+. 
expected_version="5.0"
if [[ "$addon_version" == *"$expected_version"* ]]; then
    echo "Expected addon version is enabled"
else
    enableAddon $CLUSTER_NAME
fi

mkdir -p "$GOPATH/src" "$GOPATH/bin" && sudo chmod -R 777 "$GOPATH"
error "Unable to create src under GOPATH"
cd $GOPATH/src
git clone git@github.com:IBM/ibm-csi-common.git -q -b snape2eguna
touch /root/go/src/ibm-csi-common/snapshote2e_test_result.log
cd ibm-csi-common
DIR="$(pwd)"
echo "Present working directory: $DIR"

echo "Starting snapshot basic e2e tests for vpc block storage"
export E2E_TEST_RESULT=$GOPATH/src/ibm-csi-common/snapshote2e_test_result.log
ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[snapshot\]"  ./tests/snapshote2e
cat $E2E_TEST_RESULT
echo "Finished snapshot basic e2e tests for vpc block storage"








