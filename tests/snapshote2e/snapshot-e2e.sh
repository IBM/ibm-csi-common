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

# fetchExpectedVersion() fetches the addon version to be enabled
fetchExpectedVersion() {
        # Fetches all the addon versions after 4.+ into an array addonVersions
        IFS=$'\n' read -r -d '' -a addonVersions < <( ibmcloud ks cluster addon versions | grep vpc-block-csi-driver  | awk '{print $2}' | grep -v "4." )

        # Listing the addon versions for debugging purpose
        echo "Existing addon versions"
        echo ${addonVersions[@]}

        # If there is only one addon version, we use that
        if [[ "${#addonVersions[@]}" -eq 1 ]]; then
                EXPECTED_VERSION=${addonVersions[0]}
                return 0
        fi

        # Fetching the latest addon version
        latest=0
        for addonVersion in "${addonVersions[@]}";
        do
                # Ignoring if it is beta
                if [[ "$addonVersion" == *"beta"* ]]; then
                        continue
                fi

                # Parsing the version after 5.+
                IFS=$'.' read -r -d '' -a version < <( echo "$addonVersion" )
                if [[ ${version[1]} -ge latest ]]; then
                        latest=${version[1]}
                        EXPECTED_VERSION=$addonVersion
                fi
        done

        if [[ "$EXPECTED_VERSION" == "" ]]; then
            echo "Unable to fetch expected addon version"
            return 1
        fi
}

# enableAddon() - disables the existing vpc-block-csi-driver addon and enables the 5.* addon from which snapshot is supported.
enableAddon() {
    ibmcloud ks cluster addon disable  vpc-block-csi-driver -f -c $1
    # waiting for addon to be disabled
    sleep 60s

    # Enabling addon
    ibmcloud ks cluster addon enable vpc-block-csi-driver  -c $1 --version $EXPECTED_VERSION
    sleep 60s

    # Check if controller server pod is up and running
    kubectl get pods -n kube-system | grep ibm-vpc-block-csi-controller
    if [[ $? != 0 ]]; then
        error "Driver is not enabled"
    fi

    # List all pods in kube-system - if needed for debugging
    kubectl get pods -n kube-system
}

# Print KUBECONFIG
echo "KUBECONFIG is set to $KUBECONFIG"

#Get cluster name
IN=$(kubectl config current-context)
error "Unable to fetch kubectl config"
IFS='/' read -ra CLUSTER_NAME <<< "$IN"
echo "Cluster Name - $CLUSTER_NAME"

#Fetch cluster addon version
addon_version=$(ibmcloud ks cluster addon ls -c $CLUSTER_NAME | grep vpc-block-csi-driver | awk '{print $2}')
error "Unable to fetch vpc-block-csi-driver addon version"
echo "Current vpc-block-csi-driver addon version - $addon_version"

#Fetching expected addon version
fetchExpectedVersion
error "Unable to fetch expected addon version"

#Check if the addon version is 5.+, if it is not disable the existing one, and enable 5.+. 
if [[ "$addon_version" == "$EXPECTED_VERSION" ]]; then
    echo "Expected addon version is enabled"
else
    enableAddon $CLUSTER_NAME
fi

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






