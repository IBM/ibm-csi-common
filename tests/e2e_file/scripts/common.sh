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

CUR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONF_DIR="$(dirname "$CUR_DIR")"/conf


function setKubeConfig {
    if [ -z $1 ]; then
        echo "Cluster name not specified, ${FUNCNAME[0]} skipped"
        return 0
    fi

    # Get the kube config from the `bx` cli and export KUBECONFIG for
    # your current bash session

    cluster_name=$1
    echo "Generating Kube Config through 'ibmcloud cs cluster-config $cluster_name --admin' and exporting KUBECONFIG"
    configfile=$(ibmcloud cs cluster-config $cluster_name --admin | grep ^export | cut -d '=' -f 2)
    cat $configfile
    export KUBECONFIG=$configfile

}


# Create a cluster
function cluster_create {
    set -x
    cluster_name=$1

    ibmcloud ks cluster-create-vpc-classic --name $cluster_name --vpc-id $VPC_ID --subnet-id $SUBNET_ID \
	--zone $VPC_CLUSTER_ZONE --kube-version $CLUSTER_KUBE_VERSION --flavor $VPC_FLAVOR \
        --provider gc --workers $WORKER_COUNT
    set +x
}


# Wait for cluster delete
function check_cluster_deleted {
    if [ -z $1 ]; then
        echo "Cluster name not specified, ${FUNCNAME[0]} skipped"
        return 0
    fi
    attempts=0
    set +x
    cluster_name=$1
    cluster_id=$(ibmcloud ks clusters | awk "/^$cluster_name/"'{print $2}')
    echo "Waiting for $cluster_name ($cluster_id) to be deleted..."
    while true; do
        cluster_count=$(ibmcloud ks clusters | grep  $cluster_name | wc -l | xargs)
        if [[ $cluster_count -eq 0 ]]; then
            # sometimes cluster is not deleted even though the count is 0.
            # Sleep for sometime to make sure complete clean up is done
            sleep 60
            break;
        fi
	echo "$cluster_count instances still exist"
        state=$(ibmcloud ks clusters | awk "/^$cluster_name/"'{print $3}')

        attempts=$((attempts+1))
        if [[ $state == "*_failed" ]]; then
            echo "$cluster_name ($cluster_id) entered a $state state. Exiting"
            exit 1
        fi
        if [[ $attempts -gt 120 ]]; then
            echo "$cluster_name ($cluster_id) failed to be deleted after 15 minutes. Exiting."
            # Show cluster workers state as it is helpful.
            ibmcloud ks workers $cluster_name
            exit 2
        fi
        echo "$cluster_name ($cluster_id) state == $state.  Sleeping 30 seconds"
        sleep 30
    done
    ibmcloud ks clusters
}

# Check cluster state
function check_cluster_state {
    if [ -z $1 ]; then
        echo "Cluster name not specified, ${FUNCNAME[0]} skipped"
        return 0
    fi
    attempts=0
    set +x
    cluster_name=$1
    cluster_id=$(ibmcloud ks clusters | awk "/^$cluster_name/"'{print $2}')
    echo "Waiting for $cluster_name ($cluster_id) to reach deployed state..."
    while true; do
        state=$(ibmcloud ks clusters | awk "/^$cluster_name/"'{print $3}')
        attempts=$((attempts+1))
        if [[ $state == "*_failed" ]]; then
            echo "$cluster_name ($cluster_id) entered a $state state. Exiting"
            exit 1
        # There are multiple states that equate to deployed if $state matches
        # any of them, then break out of the loop.  Without the $, normals would
        # be a valid match.
        elif [[ ${state} =~ deployed$|normal$|warning$|critical$|pending$ ]]; then
            echo "$cluster_name ($cluster_id) reached a valid state!"
            break
        fi
        if [[ $attempts -gt 120 ]]; then
            echo "$cluster_name ($cluster_id) failed to reach a valid state after 15 minutes. Exiting."
            # Show cluster workers state as it is helpful.
            ibmcloud ks workers $cluster_name
            exit 2
        fi
        echo "$cluster_name ($cluster_id) state == $state.  Sleeping 30 seconds"
        sleep 30
    done
    ibmcloud ks clusters
}

# Check cluster worker state
function check_worker_state {
    if [ -z $1 ]; then
        echo "Cluster name not specified, ${FUNCNAME[0]} skipped"
        return 0
    fi
    all_workers_good=0
    cluster_name=$1

    TIMEOUT=${WORKER_STATE_TIMEOUT:-90}
    echo "Waiting for $cluster_name workers to reach deployed state( $TIMEOUT minutes )..."
    set +e

        ibmcloud ks workers $cluster_name | grep $VPC_CLUSTER_ZONE
    set -e

    # Try for up to 90 minutes(default) for the workers to reach deployed state
    for ((i=1; i<=TIMEOUT; i++)); do
        oldifs="$IFS"
        IFS=$'\n'
            workers=($(ibmcloud ks workers $cluster_name | grep $VPC_CLUSTER_ZONE))
        IFS="$oldifs"
        worker_cnt=${#workers[@]}
        # Inspect the state of each worker
        for worker in "${workers[@]}"; do
            # Fail if any are in failed state
            state=$(echo $worker | awk '{print $4}')
            worker_id=$(echo $worker | awk '{print $1}')
            if [[ $state == "*_failed" ]]; then
                echo "$cluster_name worker $worker_id entered a $state state. Exiting"
                exit 1
            elif [[ ${state} =~ deployed$|normal$|warning$|critical$ ]]; then
                echo "$cluster_name worker $worker_id state == $state."
                # Count the number of workers in deployed state
                all_workers_good=$((all_workers_good+1))
            fi
        done
        if [[ $worker_cnt -eq $all_workers_good ]]; then
            # Break out of the 30 minute loop since all workers reach deployed state
            break
        else
            # Else sleep 60 seconds

            # Ignore failures on this command call
            ibmcloud ks workers  $cluster_name || true
            status_msg="$all_workers_good of $worker_cnt $cluster_name workers are in deployed state. Sleeping 60 seconds."
            echo "$status_msg"
            sleep 60
            all_workers_good=0
        fi
    done

    # Ignore failures on this command call
    ibmcloud ks workers $cluster_name || true
    if [[ $worker_cnt -ne $all_workers_good ]]; then
        echo "Not all $cluster_name workers reached deployed state in 40 minutes."
        return 1
    fi
    # All is good
    return 0
}

function rm_cluster {
    if [ -z $1 ]; then
        echo "Cluster name not specified, ${FUNCNAME[0]} skipped"
        return 0
    fi
    removed=1
    cluster_name=$1

    ibmcloud ks clusters

    for i in {1..3}; do
        if ibmcloud cs cluster-rm $cluster_name -f; then
            sleep 30
            # Remove old kubeconfig files aswell
            rm -rf $HOME/.bluemix/plugins/container-service/clusters/$cluster_name
            removed=0
            break
        fi
        sleep 30
    done
    return $removed
}

function bx_login {
    echo 'Logging Into IbmCloud Container Service'
    ibmcloud --version
    ibmcloud plugin list
    ibmcloud login  -r $VPC_LOGIN_REGION -a $IC_API_ENDPOINT -u $IC_LOGIN_USER -p $IC_LOGIN_PASSWORD -c $IC_ACCOUNT
    ibmcloud ks init --host $IC_HOST_EP

}

function addFullPathToCertsInKubeConfig {
    # The e2e tests expect the full path of the certs
    # to be in the kube confg. Prior to calling this function.
    # it is expected to have a `KUBECONFIG` variable defined

    CONFIG_DIR=$(dirname $KUBECONFIG)
    pushd ${CONFIG_DIR}

    for certFile in $(ls | grep -E ".*.pem"); do
        certFilePATH=${CONFIG_DIR}/${certFile}
        # Replace the certs with full path unless they already have the full path
        sed "s|[^\/]$certFile| $certFilePATH|g" $KUBECONFIG > /tmp/kubeconfig.yml;mv /tmp/kubeconfig.yml $KUBECONFIG
    done
    popd
}

function deploy_file_vpc_csi_driver {
   # Check if add-on already exists
   ibmcloud ks cluster-addons --cluster $FILE_VPC_CLUSTER | grep vpc-file-csi-driver
   retCode=$?
   if [[ $retCode != 0 ]] ; then
       echo "vpc-file-csi-driver add-on need to be enabled"
       ibmcloud ks cluster-addon-enable vpc-file-csi-driver --cluster $FILE_VPC_CLUSTER
       echo "Exit status of driver deployment command is  $?"
   else
       echo "vpc-file-csi-driver already exists"
   fi
   check_statefulset "ibm-vpc-file-csi-controller"
   check_daemonset_state "ibm-vpc-file-csi-node"
}

function check_statefulset {
   attempts=0
   pod_name=$1
   while true; do
      attempts=$((attempts+1))
      pod_status=$(kubectl get pods -n kube-system | awk "/$pod_name/"'{print $2}')
      if [   "$pod_status" = "3/3" ]; then
         echo "$pod_name is  running ."
         break
      fi
      if [[ $attempts -gt 30 ]]; then
         echo "$pod_name  were not running well."
         kubectl get pods -n kube-system| awk "/$pod_name-/"
         exit 1
      fi
      echo "$pod_name state == $pod_status  Sleeping 10 seconds"
      sleep 10
  done

}

function check_daemonset_state {
   attempts=0
   ds_name=$1
   while true; do
       attempts=$((attempts+1))
       ds_status_desired=$(kubectl get ds -n kube-system | awk "/$ds_name/"'{print $2}')
       ds_status_available=$(kubectl get ds -n kube-system | awk "/$ds_name/"'{print $6}')
       if [   "$ds_status_desired" = "$ds_status_available" ]; then
          echo "$ds_name is  running and available ds instances: $ds_status_available"
          break
       fi
       if [[ $attempts -gt 30 ]]; then
          echo "$ds_name  were not running well. Instances Desired:$deployment_status_desired, Instances Available:$deployment_status_available"
          kubectl get ds -n kube-system| awk "/$ds_name/"
          exit 1
       fi
       echo "DS:$ds_name, Desired:$ds_status_desired, Available:$ds_status_available  Sleeping 10 seconds"
       sleep 10
   done
}

function copy_secrets_to_kube_system {
    set -x
    kubectl get secrets -n kube-system | grep bluemix-default-secret
    retCode=$?
    if [[ $retCode != 0 ]] ; then

        kubectl get secret -n default default-stg-icr-io -o yaml \
	    | sed 's/namespace: default/namespace: kube-system/g'  | sed 's/name: default-stg-icr-io/name: bluemix-default-secret/g' | \
	    kubectl -n kube-system create -f -
   	kubectl get secrets -n kube-system | grep bluemix

    fi

    # Delete any existing deployments
    sh deploy/kubernetes/driver/kubernetes/delete-vpc-csi-driver.sh stage
    sed -i 's/registry.stage1.ng.bluemix.net/stg.icr.io/g' \
        deploy/kubernetes/driver/kubernetes/overlays/stage/controller-server-images.yaml \
	deploy/kubernetes/driver/kubernetes/overlays/stage/node-server-images.yaml
	echo $?
}
