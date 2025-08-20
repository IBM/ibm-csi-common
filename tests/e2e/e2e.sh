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
E2E_TEST_SETUP="$VPC_BLOCK_CSI_HOME/e2e-setup.out"
E2E_TEST_RESULT="$VPC_BLOCK_CSI_HOME/e2e-test.out"
export E2E_TEST_RESULT=$E2E_TEST_RESULT
export E2E_TEST_SETUP=$E2E_TEST_SETUP
SECRET_CREATION_WAIT=600 #seconds

rm -f $E2E_TEST_RESULT
rm -f $E2E_TEST_SETUP

IC_LOGIN="false"
PVCCOUNT="single"

UNKOWNPARAM=()
while [[ $# -gt 0 ]]; do
	key="$1"
	case $key in
		-l|--login)
		IC_LOGIN="true"
		shift
		;;
		-e|--env)
		TEST_ENV="$2"
		shift
		shift
		;;
    		-r|--region)
		REGION="$2"
		shift
		shift
		;;

		-tp| --use-trusted-profile)
		e2e_tp="$2"
		shift;
		shift
		;;

		--run-acadia)
		e2e_acadia_profile_test_case="$2"
		shift
		shift
		;;
    		*)
    		UNKOWNPARAM+=("$1")
    		shift
    		;;
	esac
done

export E2E_ZONE=$REGION-1

if [[ "$IC_LOGIN" == "true" ]]; then
	echo "Kube Config already exported!!!"
fi

if [[ "$IC_LOGIN" != "true" ]]; then
   echo "Error: Not logged into IBM Cloud!!!"
   echo "VPC-BLK-CSI-TEST: Cluster-Setup: FAILED" > $E2E_TEST_RESULT
   exit 1
fi

echo "**********VPC-Block-Volume-Tests**********" > $E2E_TEST_RESULT
echo "********** E2E Test Details **********" > $E2E_TEST_SETUP
echo -e "StartTime   : $(date "+%F-%T")" >> $E2E_TEST_SETUP

CLUSTER_DETAIL=$(kubectl get cm cluster-info -n kube-system -o jsonpath='{.data.cluster-config\.json}' |\
		 grep -v -e 'crn' -e 'master_public_url' -e 'master_url'); rc=$?
if [[ $rc -ne 0 ]]; then
	echo -e "Error       : Setup failed" >> $E2E_TEST_SETUP
	echo -e "Error       : Unable to connect to the cluster" >> $E2E_TEST_SETUP
	echo -e "Error       : Unbale to execute e2e test!"
	echo -e "VPC-BLK-CSI-TEST: VPC-Block-Volume-Tests: FAILED" >> $E2E_TEST_RESULT
	exit 1
fi

# Validate that ibm-cloud-credentials is created
wait_for_secret() {
    echo
    echo "⏳ Waiting up to ${SECRET_CREATION_WAIT}s for ibm-cloud-credentials to appear..."

    local elapsed=0
    while [[ $elapsed -lt ${SECRET_CREATION_WAIT} ]]; do
      if kubectl get secret ibm-cloud-credentials -n kube-system; then
        echo "ibm-cloud-credentials found in namespace kube-system."
        return 0
      fi

      sleep 5
      ((elapsed+=5))
    done

    echo "ibm-cloud-credentials was not created within ${SECRET_CREATION_WAIT}s."
    return 1
}

function check_trusted_profile_status {
	set -x
	expected_profile_id=""
    if [[ "$e2e_tp" == "true" ]]; then
		if [[ "$TEST_ENV" == "stage" ]]; then
            expected_profile_id=$STAGE_TRUSTED_PROFILE_ID
        else
            expected_profile_id=$PROD_TRUSTED_PROFILE_ID
        fi
		echo "************************Trusted Profile Check ***************************" >> $E2E_TEST_SETUP
        # Secret existence
        wait_for_secret
        secret_json=$(kubectl get secret ibm-cloud-credentials -n kube-system -o json)
        encoded=$(jq -r '.data["ibm-credentials.env"]' <<< "$secret_json")
        decoded=$(base64 --decode <<< "$encoded")
        profileID=$(echo $decoded | grep IBMCLOUD_PROFILEID | cut -d'=' -f3-)
        if [[ "$profileID" == "$expected_profile_id" ]]; then
            echo -e "VPC-BLOCK-CSI-TEST: USING TRUSTED_PROFILE: TRUE" >> $E2E_TEST_SETUP
			echo "***************************************************" >> $E2E_TEST_SETUP
        else
            echo -e "VPC-BLOCK-CSI-TEST: USING TRUSTED_PROFILE: FAILED" >> $E2E_TEST_SETUP
			echo "***************************************************" >> $E2E_TEST_SETUP
            exit 1
        fi
    fi
}

check_trusted_profile_status

CLUSTER_KUBE_DETAIL=$(kubectl get nodes -o jsonpath="{range .items[*]}{.metadata.name}:{.status.nodeInfo.kubeletVersion}:{.status.nodeInfo.osImage} {'\n'}"); rc=$?
echo -e "***************** Cluster Details ******************" >> $E2E_TEST_SETUP
echo -e "$CLUSTER_DETAIL" >> $E2E_TEST_SETUP
echo -e "----------------------------------------------------" >> $E2E_TEST_SETUP

echo -e "----------------------------------------------------" >> $E2E_TEST_SETUP
echo -e "$CLUSTER_KUBE_DETAIL" >> $E2E_TEST_SETUP
echo -e "----------------------------------------------------" >> $E2E_TEST_SETUP
echo -e "$DRIVER_DETAILS" >> $E2E_TEST_SETUP
echo -e "Addon Version: $CLUSTER_ADDON_VER" >> $E2E_TEST_SETUP
echo "***************************************************" >> $E2E_TEST_SETUP

err_msg1=""
err_msg2=""
DRIVER_PODS=$(kubectl get pods -n kube-system | grep 'ibm-vpc-block-csi-controlle' | grep 'Running'); rc=$?
if [[ $rc -ne 0 ]]; then
    err_msg1="Error       : Controller not active"
	echo "***************************************************" >> $E2E_TEST_SETUP
	DRIVER_DETAILS=$(kubectl describe pod -n kube-system ibm-vpc-block-csi-controller | sed -n '/Events/,$p'); 
	echo -e "\nDRIVER DETAILS = $DRIVER_DETAILS" >> $E2E_TEST_SETUP
	echo "***************************************************" >> $E2E_TEST_SETUP
else
	echo "***************************************************" >> $E2E_TEST_SETUP
    DRIVER_DETAILS=$(kubectl get pods -n kube-system ibm-vpc-block-csi-controller-0 -o jsonpath="{range .spec.containers[*]}{.name}:{.image}{'\n'}"); rc=$?
	echo -e "\nDRIVER DETAILS = $DRIVER_DETAILS" >> $E2E_TEST_SETUP
	echo "***************************************************" >> $E2E_TEST_SETUP
fi

DRIVER_PODS=$(kubectl get pods -n kube-system | grep 'ibm-vpc-block-csi-node' | grep 'Running'); rc=$?
if [[ $rc -ne 0 ]]; then
    err_msg2="Error       : Node server not active"
	echo "***************************************************" >> $E2E_TEST_SETUP
	DRIVER_DETAILS=$(kubectl describe pod -n kube-system ibm-vpc-block-csi-node | sed -n '/Events/,$p');
	echo -e "\nDRIVER DETAILS = $DRIVER_DETAILS" >> $E2E_TEST_SETUP
	echo "***************************************************" >> $E2E_TEST_SETUP
fi

if [[ -n "$err_msg1" || -n "$err_msg2" ]]; then
	echo -e "Error       : Setup failed" >> $E2E_TEST_SETUP
	[[ -n "$err_msg1" ]] && echo -e "$err_msg1" >> $E2E_TEST_SETUP
	[[ -n "$err_msg2" ]] && echo -e "$err_msg2" >> $E2E_TEST_SETUP
	echo "***************************************************" >> $E2E_TEST_SETUP
	echo -e "VPC-BLK-CSI-TEST: VPC-Block-Volume-Tests: FAILED" >> $E2E_TEST_RESULT
	exit 1
fi

set +e
# check mandatory variables
echo "Running E2E for region: [$TEST_ENV]"
echo "                  Path: `pwd`"

# E2E Execution
go clean -modcache
export GO111MODULE=on
# Install supported ginkgo version as of July 2024. Update it if necessary
go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo@v2.21.0
set +e
ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[sc\]" ./tests/e2e -- -e2e-verify-service-account=false
rc1=$?
echo "Exit status for basic volume test: $rc1"

ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[snapshot\]" ./tests/e2e -- -e2e-verify-service-account=false
rc2=$?
echo "Exit status for snapshot test: $rc2"

ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[resize\] \[pv\]" ./tests/e2e -- -e2e-verify-service-account=false
rc3=$?
echo "Exit status for resize volume test: $rc3"


set -x
VA_ADDON_VERSION=5.2
CLUSTER_ADDON_MAJOR=$(echo "$CLUSTER_ADDON_VER" | awk -F'.' '{print $1"."$2}')
compare=`echo | awk "{ print ($CLUSTER_ADDON_MAJOR >= $VA_ADDON_VERSION)?1 : 0 }"`
echo $compare
if [[ $compare -eq 1 ]]; then
	ginkgo -v --focus="\[ics-e2e\] \[volume-attachment-limit\] \[default\]" ./tests/e2e
	rc4=$?
	echo "Exit status for default attach-volume test: $rc4"

	ginkgo -v --focus="\[ics-e2e\] \[volume-attachment-limit\] \[config\]" ./tests/e2e
	rc5=$?
	echo "Exit status for configmap related attach-volume test: $rc5"
fi

#version check
version_ge() {
    [ "$(printf '%s\n' "$1" "$2" | sort -V | head -n1)" = "$2" ]
}

# Acadia Profile based tests
rc6=${rc6:-0}
if version_ge "$CLUSTER_ADDON_MAJOR" "$VA_ADDON_VERSION"; then
	if [[ "$e2e_acadia_profile_test_case" == "true" ]]; then
    	# Run Acadia profile tests for other regions only if test case flag is true
 		ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[with-sdp-profile\]" ./tests/e2e
    	rc6=$?
    	echo "Exit status for Acadia profile test (flag-enabled, other region): $rc6"
	else
    	# Skip the test if conditions are not met
    	echo -e "VPC-BLOCK-CSI-TEST-ACADIA: VPC-BLOCK-ACADIA-PROFILE-TESTS: SKIP" >> "$E2E_TEST_RESULT"
	fi
else
	# Skip the test if conditions are not met
    echo -e "VPC-BLOCK-CSI-TEST-ACADIA: VPC-BLOCK-ACADIA-PROFILE-TESTS: SKIP" >> "$E2E_TEST_RESULT"
fi

if [[ $rc1 -eq 0 && $rc2 -eq 0 && $rc3 -eq 0 && $rc4 -eq 0 && $rc5 -eq 0 && $rc6 -eq 0 ]]; then
	echo -e "VPC-BLK-CSI-TEST: VPC-Block-Volume-Tests: PASS" >> $E2E_TEST_RESULT
else
	echo -e "VPC-BLK-CSI-TEST: VPC-Block-Volume-Tests: FAILED" >> $E2E_TEST_RESULT
fi

grep  'VPC-BLK-CSI-TEST: VPC-Block-Volume-Tests: FAILED' $E2E_TEST_RESULT; rc=$?
if [[ $rc -eq 0 ]]; then
	exit 1
else
	exit 0
fi
