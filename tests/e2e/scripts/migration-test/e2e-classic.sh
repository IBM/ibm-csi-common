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
VPC_BLOCK_CSI_HOME=$GOPATH/src/github.com/IBM/ibm-csi-common
E2E_TEST_SETUP=$VPC_BLOCK_CSI_HOME/e2e-setup.out
E2E_TEST_RESULT=$VPC_BLOCK_CSI_HOME/e2e-test.out

PVCCOUNT="single"

UNKOWNPARAM=()
while [[ $# -gt 0 ]]; do
	key="$1"

	case $key in
	        -e|--env)
        ENV_REGION="$2"
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
            -p|--pvccount)
        PVCCOUNT="$2"
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

if [[ ! -z ${REGION} ]]; then
    CONFIGPATH="./tests/e2e/scripts/migration-test/kube-config/classic"
    CWDIR=$(pwd)

    if [[ ! -e ${CONFIGPATH}/kube-config-${REGION}.tar ]]; then
        echo "********** E2E POD Migration Test Details **********" > $E2E_TEST_SETUP
        echo -e "StartTime   : $(date "+%F-%T")" >> $E2E_TEST_SETUP
        echo -e "Error       : Setup failed" >> $E2E_TEST_SETUP
        echo -e "Error       : Cluster config not found ${CONFIGPATH}/kube-config-${REGION}.tar" >> $E2E_TEST_SETUP
            echo -e "Error       : Unbale to execute e2e test!" >> $E2E_TEST_SETUP
        echo -e "CLASSIC-BLK-TEST: Deployment POD migration from Node-A to Node-B: FAILED" > $E2E_TEST_RESULT
        exit 1
    fi
    if [[ ! -e ./tests/e2e/scripts/migration-test/${TESTSCRIPT}  ]]; then
        echo "********** E2E POD Migration Test Details **********" > $E2E_TEST_SETUP
        echo -e "StartTime   : $(date "+%F-%T")" >> $E2E_TEST_SETUP
        echo -e "Error       : Setup failed" >> $E2E_TEST_SETUP
        echo -e "Error       : Testscript not found ./tests/e2e/scripts/migration-test/${TESTSCRIPT}" >> $E2E_TEST_SETUP
            echo -e "Error       : Unbale to execute e2e test!" >> $E2E_TEST_SETUP
        echo -e "CLASSIC-BLK-TEST: Deployment POD migration from Node-A to Node-B: FAILED" > $E2E_TEST_RESULT
        exit 1
    fi

    mkdir -p ${CONFIGPATH}/${REGION}/
    tar -xvf ${CONFIGPATH}/kube-config-${REGION}.tar  -C ${CONFIGPATH}/${REGION}/
    chmod 700 ${CONFIGPATH}/${REGION}
    export KUBECONFIG=${CWDIR}/${CONFIGPATH#*/}/${REGION}/vpc-cluster-kube-config.yml
    echo "KUBECONFIG: $KUBECONFIG"
    if [[ ! -e ${KUBECONFIG} ]]; then
        echo "********** E2E POD Migration Test Details **********" > $E2E_TEST_SETUP
        echo -e "StartTime   : $(date "+%F-%T")" >> $E2E_TEST_SETUP
        echo -e "Error       : Setup failed" >> $E2E_TEST_SETUP
        echo -e "Error       : Cluster config not found ${KUBECONFIG}"
            echo -e "Error       : Unbale to execute e2e test!"
        echo -e "CLASSIC-BLK-TEST: Deployment POD migration from Node-A to Node-B: FAILED" > $E2E_TEST_RESULT
        exit 1
    fi
fi

CLUSTER_KUBE_DETAIL=$(kubectl get nodes -o jsonpath="{range .items[*]}{.metadata.name}:{.status.nodeInfo.kubeletVersion}:{.status.nodeInfo.osImage} {'\n'}")
CLUSTER_DETAIL=$(kubectl get cm cluster-info -n kube-system -o jsonpath='{.data.cluster-config\.json}' | grep -v crn)
if [[ $rc -ne 0 ]]; then
        echo "********** E2E POD Migration Test Details **********" > $E2E_TEST_SETUP
        echo -e "StartTime   : $(date "+%F-%T")" >> $E2E_TEST_SETUP
        echo -e "Error       : Setup failed" >> $E2E_TEST_SETUP
        echo -e "Error       : Unable to coonect to cluster"
        echo -e "Error       : Unbale to execute e2e test!"
        echo -e "CLASSIC-BLK-TEST: Deployment POD migration from Node-A to Node-B: FAILED" > $E2E_TEST_RESULT
        exit 1
fi

echo "********** E2E POD Migration Test Details **********" > $E2E_TEST_SETUP
echo -e "StartTime   : $(date "+%F-%T")" >> $E2E_TEST_SETUP
echo -e "Build       : $BUILD_NUMBER (jenkin)" >> $E2E_TEST_SETUP
echo -e "***************** Cluster Details ******************" >> $E2E_TEST_SETUP
echo -e "$CLUSTER_DETAIL" >> $E2E_TEST_SETUP
echo -e "$CLUSTER_KUBE_DETAIL" >> $E2E_TEST_SETUP
echo "***************************************************" >> $E2E_TEST_SETUP

set +e
# check mandatory variables
echo "Running E2E for region: [$ENV_REGION]"
echo "                  Path: `pwd`"
chmod 755 ./tests/e2e/scripts/migration-test/${TESTSCRIPT}
./tests/e2e/scripts/migration-test/${TESTSCRIPT} -p $PVCCOUNT | tee -a $E2E_TEST_RESULT
grep  'CLASSIC-BLK-TEST: Deployment POD migration from Node-A to Node-B: FAILED'; rc=R?
if [[ $rc -eq 0 ]]; then
	exit 1
else
	exit 0
fi
