#!/bin/bash

# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2024 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************


check_operator_enabling() {
   for (( j=0; j<60; j++ )); 
   do
       pod=$(kubectl get pods -n kube-system | grep ibm-storage-operator | awk '{print $3}')
       if [[ $pod == "Running" ]]
       then
            echo "operator pod is running"
            return 0 
       fi
       sleep 10
   done
   return 1
}

check_file_enabling() { 
   echo "waiting for 5 minutes before checking file csi driver status"
   sleep 420

   kubectl get pods -n kube-system --no-headers | grep ibm-vpc-file-csi-controller > controller-pods.log 2>&1
   IFS=$'\n' read -r -d '' -a contrllerServerPods < <( cat controller-pods.log | awk 'NR>1' && printf '\0' )
   if [[ $contrllerServerPods == "" ]];
   then
      echo "No controller server pods found"
      return 1
   fi
   for controllerPod in "${contrllerServerPods[@]}";
   do
      podState=$(echo $controllerPod | awk '{print $3}')
      if [[ $podState != "Running" ]]
      then
            echo "$controllerPod is not in running state"
            return 1
      fi
   done

   kubectl get pods -n kube-system --no-headers | grep ibm-vpc-file-csi-node > node-server-pods.log 2>&1
   IFS=$'\n' read -r -d '' -a nodeServerPods < <( cat node-server-pods.log | awk 'NR>1' && printf '\0' )
   if [[ $nodeServerPods == "" ]];
   then
      echo "No node server pods found"
      return 1
   fi
   for nodeServerPod in "${nodeServerPods[@]}";
   do
      podState=$(echo $nodeServerPod | awk '{print $3}')
      if [[ $podState != "Running" ]]
      then 
            echo "$nodeServerPod is not in running state"
            return 1
      fi
   done
}

check_file_disabling() {
   sleep 60
   failure=0
   pods=$(kubectl get pods -n kube-system --no-headers | grep ibm-vpc-file)
   if [[ $pods == "" ]];
   then
      echo "File csi driver pods deleted"
   else
      echo "File csi driver pods not deleted"
      failure=1
   fi

   deploy=$(kubectl get deploy -n kube-system | grep ibm-vpc-file-csi-controller)
   if [[ $deploy == "" ]];
   then 
      echo "File controller deployment deleted"
   else
      echo "File controller deployment not deleted"
      failure=1
   fi

   ds=$(kubectl get ds -n kube-system | grep ibm-vpc-file-csi-node)
   if [[ $ds == "" ]];
   then 
      echo "Node server ds deleted"
   else
      echo "Node server ds not deleted"
      failure=1
   fi

   clusterrole=$(kubectl get clusterroles | grep vpc-file)
   if [[ $clusterrole == "" ]];
   then 
      echo "Clusterroles for file deleted"
   else
      echo "Clusterroles for file not deleted"
      failure=1
   fi

   clusterrolebinding=$(kubectl get clusterrolebindings | grep vpc-file)
   if [[ $clusterrolebinding == "" ]];
   then 
      echo "ClusterroleBindings for file deleted"
   else
      echo "ClusterroleBindings for file not deleted"
      failure=1
   fi

   serviceaccount=$(kubectl get sa -n kube-system | grep ibm-vpc-file)
   if [[ $serviceaccount == "" ]];
   then 
      echo "Serviceaccounts for file deleted"
   else
      echo "Serviceaccounts for file not deleted"
      failure=1
   fi

   return $failure
}

check_operator_disabling() {
   echo "waiting for a minute before checking operator status"
   sleep 60
   failure=0
   pods=$(kubectl get pods -n kube-system --no-headers | grep ibm-storage-operator)
   if [[ $pods == "" ]];
   then
      echo "operator pods deleted"
   else
      echo "operator pods not deleted"
      failure=1
   fi

   deploy=$(kubectl get deploy -n kube-system | grep ibm-storage-operator)
   if [[ $deploy == "" ]];
   then 
      echo "operator deployment deleted"
   else
      echo "operator deployment not deleted"
      failure=1
   fi

   crd=$(kubectl get crd | grep vpcfilecsidrivers)
   if [[ $crd == "" ]];
   then 
      echo "vpcfilecsidrivers crd deleted"
   else
      echo "vpcfilecsidrivers crd not deleted"
      failure=1
   fi

   clusterrole=$(kubectl get clusterroles | grep ibm-storage-operator-role)
   if [[ $clusterrole == "" ]];
   then 
      echo "Clusterroles for operator deleted"
   else
      echo "Clusterroles for operator not deleted"
      failure=1
   fi

   clusterrolebinding=$(kubectl get clusterrolebindings | grep ibm-storage-operator-rolebinding)
   if [[ $clusterrolebinding == "" ]];
   then 
      echo "ClusterroleBindings for operator deleted"
   else
      echo "ClusterroleBindings for operator not deleted"
      failure=1
   fi

   serviceaccount=$(kubectl get sa -n kube-system | grep ibm-storage-operator)
   if [[ $serviceaccount == "" ]];
   then 
      echo "Serviceaccounts for operator deleted"
   else
      echo "Serviceaccounts for operator not deleted"
      failure=1
   fi

   return $failure
}

IBM_STORAGE_OPERATOR_HOME="$GOPATH/src/github.com/IBM/ibm-csi-common"
#IBM_STORAGE_OPERATOR_HOME=$(pwd)
E2E_TEST_SETUP="$IBM_STORAGE_OPERATOR_HOME/e2e-setup.out"
E2E_TEST_RESULT="$IBM_STORAGE_OPERATOR_HOME/e2e-test.out"
export E2E_TEST_RESULT=$E2E_TEST_RESULT
export E2E_TEST_SETUP=$E2E_TEST_SETUP

rm -f "$E2E_TEST_RESULT"
rm -f "$E2E_TEST_SETUP"

IC_LOGIN="false"

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

export E2E_ZONE=$REGION-1

if [[ "$IC_LOGIN" == "true" ]]; then
	echo "Kube Config already exported!!!"
fi

if [[ "$IC_LOGIN" != "true" ]]; then
   echo "Error: Not logged into IBM Cloud!!!"
   echo "IBM-STORAGE-OPERATOR-TEST: Cluster-Setup: FAILED" >> "$E2E_TEST_SETUP"
   exit 1
fi

echo "**********IBM-Storage-Operator-Tests**********" >> "$E2E_TEST_RESULT"
echo "********** E2E Test Details **********" >> "$E2E_TEST_RESULT"
echo -e "StartTime   : $(date "+%F-%T")" >> "$E2E_TEST_RESULT"

CLUSTER_DETAIL=$(kubectl get cm cluster-info -n kube-system -o jsonpath='{.data.cluster-config\.json}' |\
		 grep -v -e 'crn' -e 'master_public_url' -e 'master_url'); rc=$?

if [[ $rc -ne 0 ]]; then
	echo -e "Error       : Setup failed" >> "$E2E_TEST_SETUP"
	echo -e "Error       : Unable to connect to the cluster" >> "$E2E_TEST_SETUP"
	echo -e "Error       : Unbale to execute e2e test!"
	echo -e "IBM-Storage-Operator-Test: FAILED" >> "$E2E_TEST_RESULT"
	exit 1
fi

CLUSTER_KUBE_DETAIL=$(kubectl get nodes -o jsonpath="{range .items[*]}{.metadata.name}:{.status.nodeInfo.kubeletVersion}:{.status.nodeInfo.osImage} {'\n'}"); rc=$?
echo -e "***************** Cluster Details ******************" >> "$E2E_TEST_SETUP"
echo -e "$CLUSTER_DETAIL" >> "$E2E_TEST_SETUP"
echo -e "----------------------------------------------------" >> "$E2E_TEST_SETUP"

echo -e "----------------------------------------------------" >> "$E2E_TEST_SETUP"
echo -e "$CLUSTER_KUBE_DETAIL" >> "$E2E_TEST_SETUP"
echo -e "----------------------------------------------------" >> "$E2E_TEST_SETUP"

set -x

# Test operator enablement and disablement - deploy and cleanup of resources
if [[ $OPERATOR_ADDON_VERSION == "default" ]]; then
	ibmcloud ks cluster addon enable ibm-storage-operator -c $CLUSTER_NAME
else
   ibmcloud ks cluster addon enable ibm-storage-operator -c $CLUSTER_NAME --version $OPERATOR_ADDON_VERSION
fi

check_operator_enabling; rc=$?
if [[ $rc -ne 0 ]]; then
   echo -e "IBM STORAGE OPERATOR:  Addon enable: FAIL" >> "$E2E_TEST_RESULT"
	echo -e "IBM-Storage-Operator-Test: FAILED" >> "$E2E_TEST_RESULT"
	exit 1
else
   echo "IBM STORAGE OPERATOR:  Addon enable: PASS"
   echo -e "IBM STORAGE OPERATOR:  Addon enable: PASS" >> "$E2E_TEST_RESULT"
fi


if [[ $FILE_ADDON_VERSION == "default" ]]; then
	ibmcloud ks cluster addon enable vpc-file-csi-driver -c $CLUSTER_NAME
else
	ibmcloud ks cluster addon enable vpc-file-csi-driver -c $CLUSTER_NAME --version $FILE_ADDON_VERSION
fi

check_file_enabling; rc=$?
if [[ $rc -ne 0 ]]; then
   echo -e "IBM STORAGE OPERATOR:  File csi driver addon enable: FAIL" >> "$E2E_TEST_RESULT"
	echo -e "IBM-Storage-Operator-Test: FAILED" >> "$E2E_TEST_RESULT"
	exit 1
else 
   echo -e "IBM STORAGE OPERATOR:  File csi driver addon enable: PASS" >> "$E2E_TEST_RESULT"
fi

echo "y" | ibmcloud ks cluster addon disable ibm-storage-operator -c $CLUSTER_NAME; rc=$?
if [[ $rc -eq 0 ]]; then
	echo -e "IBM STORAGE OPERATOR:  Disabling operator with file addon enabled should fail: FAIL" >> "$E2E_TEST_RESULT"
	exit 1
else
	echo -e "IBM STORAGE OPERATOR:  Disabling operator with file addon enabled should fail: PASS" >> "$E2E_TEST_RESULT"
fi

ibmcloud ks cluster addon disable vpc-file-csi-driver -c $CLUSTER_NAME -f
check_file_disabling; rc=$?
if [[ $rc -ne 0 ]]; then
	echo -e "IBM STORAGE OPERATOR:  VPC File csi driver addon disable: FAIL" >> "$E2E_TEST_RESULT"
	exit 1
else
	echo -e "IBM STORAGE OPERATOR:  VPC File csi driver addon disable: PASS" >> "$E2E_TEST_RESULT"
fi


ibmcloud ks cluster addon disable ibm-storage-operator -c $CLUSTER_NAME -f
check_operator_disabling; rc=$?
if [[ $rc -ne 0 ]]; then
	echo -e "IBM STORAGE OPERATOR:  IBM Storage operator addon disable: FAIL" >> "$E2E_TEST_RESULT"
	exit 1
else
	echo -e "IBM STORAGE OPERATOR:  IBM Storage operator addon disable: PASS" >> "$E2E_TEST_RESULT"
fi

if [[ $FILE_ADDON_VERSION == "default" ]]; then
	echo "N" | ibmcloud ks cluster addon enable vpc-file-csi-driver -c $CLUSTER_NAME; rc=$?
else
	echo "N" | ibmcloud ks cluster addon enable vpc-file-csi-driver -c $CLUSTER_NAME --version $FILE_ADDON_VERSION; rc=$?
fi

if [[ $rc -eq 0 ]]; then
	echo -e "IBM STORAGE OPERATOR:  Enable file csi driver with operator disabled should fail: FAIL" >> "$E2E_TEST_RESULT"
	exit 1
else
	echo -e "IBM STORAGE OPERATOR:  Enable file csi driver with operator disabled  should fail: PASS" >> "$E2E_TEST_RESULT"
fi

#
if [[ $FILE_ADDON_VERSION == "default" ]]; then
	ibmcloud ks cluster addon enable vpc-file-csi-driver -c $CLUSTER_NAME -y; rc=$?
else
	ibmcloud ks cluster addon enable vpc-file-csi-driver -c $CLUSTER_NAME --version $FILE_ADDON_VERSION -y; rc=$?
fi
check_operator_enabling; rc=$?
if [[ $rc -ne 0 ]]; then
   echo -e "IBM STORAGE OPERATOR:  Dependent addon enable - Operator addon enable: FAIL" >> "$E2E_TEST_RESULT"
	echo -e "IBM-Storage-Operator-Test: FAILED" >> "$E2E_TEST_RESULT"
	exit 1
else 
    echo -e "IBM STORAGE OPERATOR:  Dependent addon enable - Operator addon enable: PASS" >> "$E2E_TEST_RESULT"
fi
check_file_enabling; rc=$?
if [[ $rc -ne 0 ]]; then
   echo -e "IBM STORAGE OPERATOR:  Dependent addon enable - File CSI driver addon enable: FAIL" >> "$E2E_TEST_RESULT"
	echo -e "IBM-Storage-Operator-Test: FAILED" >> "$E2E_TEST_RESULT"
	exit 1
else 
   echo -e "IBM STORAGE OPERATOR:  Dependent addon enable - File CSI driver enable: PASS" >> "$E2E_TEST_RESULT"
fi

set +e
# check mandatory variables
echo "Running E2E for region: [$TEST_ENV]"
echo "                  Path: `pwd`"

# E2E Execution
go clean -modcache
export GO111MODULE=on
go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo@v2.1.6
ginkgo run -v  ./tests/e2e
rc2=$?
if [[ $rc -ne 0 ]]; then
   echo -e "IBM STORAGE OPERATOR:  EIT feature tests: FAIL" >> "$E2E_TEST_RESULT"
	echo -e "IBM-Storage-Operator-Test: FAILED" >> "$E2E_TEST_RESULT"
	exit 1
else
   echo -e "IBM STORAGE OPERATOR:  EIT feature tests: PASS" >> "$E2E_TEST_RESULT"
fi
set +e

