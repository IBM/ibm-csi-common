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
VPC_FILE_CSI_HOME="$GOPATH/src/github.com/IBM/ibm-csi-common"
E2E_TEST_SETUP="$VPC_FILE_CSI_HOME/e2e-setup.out"
E2E_TEST_RESULT="$VPC_FILE_CSI_HOME/e2e-test.out"

IC_LOGIN="false"

echo "1:" "$1"
echo "1:" "$2"

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
	    	-t|--testscript)
		TESTSCRIPT="$2"
		shift
		shift
    		;;
    		*)
    		UNKOWNPARAM+=("$1")
    		shift
    		;;
	esac
done

rm -f $E2E_TEST_RESULT
rm -f $E2E_TEST_SETUP

if [[ "$IC_LOGIN" == "true" ]]; then
	# ./tests/e2efile/setup.sh
	echo "Kube Config already exported!!!"
fi

if [[ "$IC_LOGIN" != "true" ]]; then
    CONFIGPATH="./tests/e2efile/scripts/statefulset-test/kube-config/vpc"
    CWDIR=$(pwd)

    if [[ ! -e ${CONFIGPATH}/kube-config-${REGION}.tar ]]; then
    	echo "********** E2E Statefulset Test Details **********" > $E2E_TEST_SETUP
    	echo -e "StartTime   : $(date "+%F-%T")" >> $E2E_TEST_SETUP
    	echo -e "Error       : Setup failed" >> $E2E_TEST_SETUP
    	echo -e "Error       : Cluster config not found ${CONFIGPATH}/kube-config-${REGION}.tar" >> $E2E_TEST_SETUP
    	echo -e "Error       : Unbale to execute e2e test!" >> $E2E_TEST_SETUP
    	echo -e "VPC-FILE-CSI-TEST: Statefulset test: FAILED" > $E2E_TEST_RESULT
    	exit 1
    fi
    if [[ ! -e ./tests/e2efile/scripts/statefulset-test/${TESTSCRIPT}  ]]; then
    	echo "********** E2E Statefulset Test Details **********" > $E2E_TEST_SETUP
    	echo -e "StartTime   : $(date "+%F-%T")" >> $E2E_TEST_SETUP
    	echo -e "Error       : Setup failed" >> $E2E_TEST_SETUP
    	echo -e "Error       : Testscript not found ./tests/e2efile/scripts/statefulset-test/${TESTSCRIPT}" >> $E2E_TEST_SETUP
    	echo -e "Error       : Unbale to execute e2e test!" >> $E2E_TEST_SETUP
    	echo -e "VPC-FILE-CSI-TEST: Statefulset test: FAILED" > $E2E_TEST_RESULT
    	exit 1
    fi

    mkdir -p ${CONFIGPATH}/${REGION}/
    tar -xvf ${CONFIGPATH}/kube-config-${REGION}.tar  -C ${CONFIGPATH}/${REGION}/
    chmod 700 ${CONFIGPATH}/${REGION}
    export KUBECONFIG=${CWDIR}/${CONFIGPATH#*/}/${REGION}/vpc-cluster-kube-config.yml
    echo "KUBECONFIG: $KUBECONFIG"
    if [[ ! -e ${KUBECONFIG} ]]; then
        echo "********** E2E StatefulSet Test Details **********" > $E2E_TEST_SETUP
        echo -e "StartTime   : $(date "+%F-%T")" >> $E2E_TEST_SETUP
        echo -e "Error       : Setup failed" >> $E2E_TEST_SETUP
        echo -e "Error       : Cluster config not found ${KUBECONFIG}"
        echo -e "Error       : Unbale to execute e2e test!"
        echo -e "VPC-FILE-CSI-TEST: StatefulSet Test: FAILED" > $E2E_TEST_RESULT
        exit 1
    fi
fi

echo "********** E2E POD Migration Test Details **********" > $E2E_TEST_SETUP
echo -e "StartTime   : $(date "+%F-%T")" >> $E2E_TEST_SETUP

CLUSTER_DETAIL=$(kubectl get cm cluster-info -n kube-system -o jsonpath='{.data.cluster-config\.json}' |\
		 grep -v -e 'crn' -e 'master_public_url' -e 'master_url'); rc=$?
if [[ $rc -ne 0 ]]; then
	echo -e "Error       : Setup failed" >> $E2E_TEST_SETUP
	echo -e "Error       : Unable to connect to the cluster" >> $E2E_TEST_SETUP
	echo -e "Error       : Unbale to execute e2e test!"
	echo -e "VPC-FILE-CSI-TEST: Statefulset Test: FAILED" > $E2E_TEST_RESULT
	exit 1
fi

echo "********** E2E Statefulset Test Details **********" > $E2E_TEST_SETUP
echo -e "***************** Cluster Details ******************" >> $E2E_TEST_SETUP
echo -e "$CLUSTER_DETAIL" >> $E2E_TEST_SETUP
echo -e "----------------------------------------------------" >> $E2E_TEST_SETUP

err_msg1=""
err_msg2=""
DRIVER_PODS=$(kubectl get pods -n kube-system | grep 'ibm-vpc-file-csi-controlle' | grep 'Running'); rc=$?
if [[ $rc -ne 0 ]]; then
    err_msg1="Error       : Controller not active"
else
   	echo "***************************************************" >> $E2E_TEST_SETUP
    DRIVER_DETAILS=$(kubectl get deployment -n kube-system ibm-vpc-file-csi-controller -o jsonpath="{range .spec.template.spec.containers[*]}{.name}:{.image}{'\n'}"); rc=$?
	echo -e "\nDRIVER DETAILS = $DRIVER_DETAILS" >> $E2E_TEST_SETUP
	echo "***************************************************" >> $E2E_TEST_SETUP
fi

DRIVER_PODS=$(kubectl get pods -n kube-system | grep 'ibm-vpc-file-csi-node' | grep 'Running'); rc=$?
if [[ $rc -ne 0 ]]; then
    err_msg2="Error       : Node server not active"
fi

if [[ -n "$err_msg1" || -n "$err_msg2" ]]; then
	echo -e "Error       : Setup failed" >> $E2E_TEST_SETUP
	[[ -n "$err_msg1" ]] && echo -e "$err_msg1" >> $E2E_TEST_SETUP
	[[ -n "$err_msg2" ]] && echo -e "$err_msg2" >> $E2E_TEST_SETUP
	echo "***************************************************" >> $E2E_TEST_SETUP
	echo -e "VPC-FILE-CSI-TEST: Deployment POD migration from Node-A to Node-B: FAILED" > $E2E_TEST_RESULT
	exit 1
fi
echo -e "$CLUSTER_KUBE_DETAIL" >> $E2E_TEST_SETUP
echo -e "----------------------------------------------------" >> $E2E_TEST_SETUP
echo -e "$DRIVER_DETAILS" >> $E2E_TEST_SETUP
echo -e "Addon Version: $CLUSTER_ADDON_VER" >> $E2E_TEST_SETUP
echo "***************************************************" >> $E2E_TEST_SETUP

set +e
# check mandatory variables
echo "Running E2E for region: [$TEST_ENV]"
echo "                  Path: `pwd`"
chmod 755 ./tests/e2efile/scripts/statefulset-test/${TESTSCRIPT}
echo "Info: ./tests/e2efile/scripts/statefulset-test/${TESTSCRIPT}"
./tests/e2efile/scripts/statefulset-test/${TESTSCRIPT} | tee -a $E2E_TEST_RESULT
grep  'VPC-FILE-CSI-TEST: Statefulset test: FAILED'; rc=$?
if [[ $rc -eq 0 ]]; then
	exit 1
else
	exit 0
fi
