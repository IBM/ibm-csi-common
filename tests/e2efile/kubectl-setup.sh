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

function icLogin {
    if [[ "$ENV_REGION" = "dev" || "$ENV_REGION" = "prestage" || "$ENV_REGION" = "stage" ]]; then
        export ENDPOINT="https://test.cloud.ibm.com"
    else
        export ENDPOINT="https://cloud.ibm.com"
    fi
    echo "Info: Loging into IBM Cloud..."
    ibmcloud login -a "$ENDPOINT" --apikey "$IC_API_KEY" --no-region || return 1
    echo "Info: Logged into IBM Cloud"
}

function initKSEndpoint {
    DEV_ARMADA_API_ENDPOINT="https://containers.dev.cloud.ibm.com/"
    PRESTAGE_ARMADA_API_ENDPOINT="https://containers.pretest.cloud.ibm.com"
    STAGE_ARMADA_API_ENDPOINT="https://containers.test.cloud.ibm.com"
    ARMADA_API_ENDPOINT="https://containers.cloud.ibm.com"

    case $ENV_REGION in
        dev)
            export ARMADA_API_ENDPOINT="${DEV_ARMADA_API_ENDPOINT}"
            ;;
        prestage)
            export ARMADA_API_ENDPOINT="${PRESTAGE_ARMADA_API_ENDPOINT}"
            ;;
        stage)
            export ARMADA_API_ENDPOINT="${STAGE_ARMADA_API_ENDPOINT}"
            ;;
	prod)
	    export ARMADA_API_ENDPOINT="https://containers.cloud.ibm.com"
            ;;
	*)
	    echo "Info: dev / prestage / stage / prod"
            ;;
    esac
    ibmcloud ks init --host "$ARMADA_API_ENDPOINT" || return 1
}

function getKubeconfig {
    echo "Info: Exporting kube config..."
    ibmcloud ks cluster config --admin --cluster $IC_CLUSTER_ID || return 1
    echo "Info: Exported kube config..."
}

icLogin || exit 1
initKSEndpoint || exit 1
getKubeconfig || exit 1
exit 0
