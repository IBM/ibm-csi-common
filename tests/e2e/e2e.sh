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
    		*)
    		UNKOWNPARAM+=("$1")
    		shift
    		;;
	esac
done

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
go get -u github.com/onsi/ginkgo/ginkgo

set +e
ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[sc\]" ./tests/e2e | tee -a block-vpc-csi-ginkgo-log.txt
rc1=$?
echo "Exit status for basic volume test: $rc1"

ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[resize\] \[pv\]" ./tests/e2e | tee -a block-vpc-csi-volume-resize-ginkgo-log.txt
rc3=$?
echo "Exit status for resize volume test: $rc3"

set -x
SNAP_ADDON_VERSION=5.0
compare=`echo | awk "{ print ($CLUSTER_ADDON_VER >= $SNAP_ADDON_VERSION)?1 : 0 }"`
echo $compare
if [[ $compare -eq 1 ]]; then
	ginkgo -v -nodes=1 --focus="\[ics-e2e\] \[snapshot\]" ./tests/e2e | tee -a block-vpc-csi-snapshot-log.txt
	rc2=$?
	echo "Exit status for snapshot test: $rc2"
fi

set -e
if [[ $rc1 -eq 0 && $rc2 -eq 0 && $rc3 -eq 0 ]]; then
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
