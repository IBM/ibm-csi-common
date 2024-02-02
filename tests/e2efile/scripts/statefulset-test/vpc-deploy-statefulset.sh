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
IT_COUNT=900
EXIT_CODE=0
NAMESPACE="ics-e2e"
BASEDIR=$(dirname "$0")

STATEFULSETSPEC="./tests/_file/scripts/statefulset-test/specs/vpc/ics-e2e-tester-statefulset.yaml"
PODTIMES=()
PVLIST=""

export CLEANUP_ON_FAILURE=${CLEANUP_ON_FAILURE:-"yes"}
export STATEFULSET_REPLICA_COUNT=${STATEFULSET_REPLICA_COUNT:-3}

# ###############################
# Cleanup Function
# ###############################
function doCleanup {
	exit_code=$1
	retcode=0

	if [[ "$CLEANUP_ON_FAILURE" == "no" && $exit_code -ne 0 ]]; then
		kubectl logs -n kube-system -c iks-vpc-file-driver ibm-vpc-file-csi-controller-0 > ./blk-controller.log
		kubectl logs -n kube-system -c csi-attacher ibm-vpc-file-csi-controller-0 > ./blk-attacher.log
		node_servers=$(kubectl get pods -n kube-system -l "app=ibm-vpc-file-csi-driver" | grep 'csi-node'  | awk '{ print $1 }')
		for nodeSrvr in $node_servers; do
			kubectl logs -n kube-system -c iks-vpc-file-node-driver $nodeSrvr > ./blk-node-${nodeSrvr}.log
		done
		return 1
	fi

	it=1
	echo "[`date`] Delete Statefulset Start"
	kubectl delete -n $NAMESPACE -f $STATEFULSETSPEC; rc=$?
	echo "[`date`] Wait Statefulset Delete"
	while [[ $it -gt 0 && $it -le $IT_COUNT ]]; do
		active_pods=$(kubectl get pod -n $NAMESPACE -l "app=nginx-vpc-file"  -o name)
		if [[ -n "$active_pods" ]]; then
			echo "[`date`] Wait Statefulset's pods delete... [${it}/${IT_COUNT}]"
			(( it += 1 ))
			sleep 3
		else
			it=-1
		fi
	done
	if [[ $it -ge $IT_COUNT ]]; then
		echo "[`date`] Wait Statefulset's pods Delete TIMEOUT"
		return 1
	fi
	echo "[`date`] Delete Statefulset End"

	errc=0
	while [[ $errc -ge 0 && $errc -lt 10 ]]; do
		pvlist=$(kubectl get pvc -n $NAMESPACE -l "app=nginx-vpc-file" -o jsonpath="{range .items[*]}{.metadata.name}:{.spec.volumeName}{'\n'}"); rc=$?
		if [[ $rc -ne 0 ]]; then
			echo "Warning: $pvlist"
			(( errc += 1 ))
			if [[ "$pvlist" == *timeout* ]]; then
				sleep 3
			else
				sleep 2
			fi
		else
			errc=-1
		fi
	done
	if [[ $errc -ge 0 ]]; then
		return 1
	fi

	ind=0
	pvArray[0]=""
	for pv in $pvlist; do
		pvArray[$ind]="$pv"
		(( ind += 1 ))
	done

	echo "[`date`] Delete PVCs Start"
	kubectl delete pvc -n $NAMESPACE -l "app=nginx-vpc-file"; rc=$?
	echo "[`date`] Delete PVCs End"

	echo "[`date`] Wait PV Delete Start"
	SECONDS=0
	it=1
	pvCount=${#pvArray[@]}
	while [[ $it -gt 0 && $it -le $IT_COUNT ]]; do
		success=0
		for (( ind = 0; ind < pvCount; ind ++ )); do
			pvname=${pvArray[$ind]}
			if [[ -z "$pvname" ]]; then
				continue
			fi
			pvname=${pvname##*:};
			success=1
			phase=$(2>&1 kubectl get pv $pvname -o jsonpath="{.status.phase}"); rc=$?
			if [[ $rc == 0 ]]; then
				echo "[`date`] ${pvArray[$ind]}:${phase} [${it}/${IT_COUNT}]"
			else
				echo "[`date`] ${pvArray[$ind]}:ERROR    [${it}/${IT_COUNT}]"
				echo "         $phase"
			fi
			if [[ "$phase" == *NotFound*$pvname* ]]; then
				pvArray[$ind]=""
			fi
		done
		if [[ $success -eq 0 ]]; then
			it=-1
		else
			(( it += 1 ))
			sleep 2
		fi
	done
	duration=$SECONDS
	echo "VPC-FILE-CSI-TEST: PV Deletion: PV Count: $pvCount Time: $(($duration / 60)) minutes and $(($duration % 60)) seconds elapsed."
	for pv in $pvlist; do
		echo "VPC-FILE-CSI-TEST:              PV: $pv"
		if [[ -z "$PVLIST" ]]; then
		   PVLIST="${pv##*:}"
		else
		   PVLIST="${PVLIST},${pv##*:}"
                fi
	done
	if [[ $it -ge $IT_COUNT ]]; then
		echo "[`date`] Wait PV Delete TIMEOUT"
		retcode=1
	fi
	echo "[`date`] Wait PV Delete End"

	echo "[`date`] Delete Namespace [$NAMESPACE]"
	kubectl delete namespace $NAMESPACE
	echo "[`date`] Deleted Namespace [$NAMESPACE]"

	kubectl logs -n kube-system -c iks-vpc-file-driver ibm-vpc-file-csi-controller-0 > ./blk-controller.log
	kubectl logs -n kube-system -c csi-attacher ibm-vpc-file-csi-controller-0 > ./blk-attacher.log
	node_servers=$(kubectl get pods -n kube-system -l "app=ibm-vpc-file-csi-driver" | grep 'csi-node'  | awk '{ print $1 }')
	for nodeSrvr in $node_servers; do
		kubectl logs -n kube-system -c iks-vpc-file-node-driver $nodeSrvr > ./blk-node-${nodeSrvr}.log
	done
	return $retcode
}

# ###############################
# Main Function
# ###############################
echo "E2E Test: Statefulset test with ${STATEFULSET_REPLICA_COUNT} replicas and each replica with 2 volumes"

seed=$(date +%s); tag=$(( $seed % 10000 ))
NAMESPACE=${NAMESPACE}-${tag}
echo "[`date`] Create Namespace [$NAMESPACE]"
kubectl create namespace $NAMESPACE; rc=$?
if [[ $rc -ne 0 ]]; then
	echo "VPC-FILE-CSI-TEST: Unable to create NS $NAMESPACE"
	echo "VPC-FILE-CSI-TEST: Statefulset test: FAILED"
	exit 1
fi
echo "[`date`] Created Namespace [$NAMESPACE]"

# Update replicas count in ics-e2e-tested-statefulset.yaml
sed -i "s/REPLICA_COUNT/${STATEFULSET_REPLICA_COUNT}/g" $STATEFULSETSPEC
echo "[`date`] Create Statefulset Start"
kubectl create -n $NAMESPACE -f $STATEFULSETSPEC; rc=$?
SECONDS=0
if [[ $rc -ne 0 ]]; then
	echo "VPC-FILE-CSI-TEST: Unable to create Statefulset"
	echo "VPC-FILE-CSI-TEST: Statefulset test: FAILED"
	exit 1
fi
echo "[`date`] Create Statefulset End"

echo "[`date`] Wait PVC Bound Start"
it=1
errc=0
while [[ $it -gt 0 && $it -le $IT_COUNT ]]; do
	success=0
        pvclist=$(2>&1 kubectl get pvc -n $NAMESPACE -l "app=nginx-vpc-file" -o jsonpath="{range .items[*]}{.metadata.namespace}:{.metadata.name}:{.status.phase}{'\n'}"); rc=$?
	if [[ $rc -ne 0 ]]; then
		if [[ "$pvclist" == *timeout* ]]; then
			echo "TIMEOUT: $pvclist"
			(( errc += 1 ))
			if [[ $errc -gt  10 ]]; then
				echo "VPC-FILE-CSI-TEST: Statefulset test: FAILED"
				exit 1
			fi
			sleep 3
			continue
		fi
	fi
	errc=0
	if [[ -z "$pvclist" ]]; then
		echo "PVC not found!"
		break
	fi
	for pvc in $pvclist; do
    		echo "[`date`] $pvc [${it}/${IT_COUNT}]"
    		phase=${pvc##*:}
    		if [[ "$phase" != "Bound" ]]; then
			success=1
    		fi
	done
	if [[ $success -eq 0 ]]; then
		it=-1
	else
		(( it += 1 ))
		sleep 2
	fi
done
duration=$SECONDS
pvcCount=0
for pvc in $pvclist; do
		(( pvcCount += 1 ))
done
echo "VPC-FILE-CSI-TEST: PVC Creation: PVC Count: $pvcCount Time: $(($duration / 60)) minutes and $(($duration % 60)) seconds elapsed."
for pvc in $pvclist; do
	echo "VPC-FILE-CSI-TEST:               PVC: $pvc"
done

if [[ $it -ge $IT_COUNT ]]; then
	echo "[`date`] Wait PVC Bound TIMEOUT"
	EXIT_CODE=1
fi
echo "[`date`] Wait PVC Bound End"
if [[ -z "$pvclist" ]]; then
	EXIT_CODE=1
fi

if [[ $EXIT_CODE -ne 0 ]]; then
	echo "[`date`] Delete Statefulset Start"
	kubectl delete -n $NAMESPACE -f $STATEFULSETSPEC; rc=$?
	echo "[`date`] Delete PVCs Start"
	kubectl delete pvc -n $NAMESPACE -l "app=nginx-vpc-file"; rc=$?
	echo "[`date`] Delete PVCs End"
	echo "[`date`] Delete Namespace [$NAMESPACE]"
	kubectl delete namespace $NAMESPACE
	echo "[`date`] Deleted Namespace [$NAMESPACE]"

        kubectl logs -n kube-system -c iks-vpc-file-driver ibm-vpc-file-csi-controller-0 > ./blk-controller.log
	kubectl logs -n kube-system -c csi-attacher ibm-vpc-file-csi-controller-0 > ./blk-attacher.log
	node_servers=$(kubectl get pods -n kube-system -l "app=ibm-vpc-file-csi-driver" | grep 'csi-node'  | awk '{ print $1 }')
	for nodeSrvr in $node_servers; do
		kubectl logs -n kube-system -c iks-vpc-file-node-driver $nodeSrvr > ./blk-node-${nodeSrvr}.log
	done
	echo "VPC-FILE-CSI-TEST: Statefulset test: FAILED"
	exit $EXIT_CODE
fi

echo "[`date`] Wait Statefulset's pods Running Start"
it=1
#SECONDS=0
errc=0
while [[ $it -gt 0 && $it -le $IT_COUNT ]]; do
	success=0
        podlist=$(2>&1 kubectl get pods -n $NAMESPACE -l "app=nginx-vpc-file" -o jsonpath="{range .items[*]}{.metadata.name}:{.spec.nodeName}:{.status.phase}{'\n'}"); rc=$?
	if [[ $rc -ne 0 ]]; then
		if [[ "$podlist" == *timeout* ]]; then
			echo "TIMEOUT: $podlist"
			(( errc += 1 ))
			if [[ $errc -gt 10 ]]; then
				echo "VPC-FILE-CSI-TEST: Statefulset test: FAILED"
				exit 1
			fi
			sleep 3
			continue
		fi
	fi
	errc=0
	if [[ -z "$podlist" ]]; then
		echo "Error: Statefulset's pods not found!"
		break
	fi
	for pod in $podlist; do
    		echo "[`date`] $pod [${it}/${IT_COUNT}]"
    		phase=${pod##*:}
    		if [[ "$phase" != "Running" ]]; then
			success=1
    		fi
	done
	if [[ $success -eq 0 ]]; then
		it=-1
	else
		(( it += 1 ))
		sleep 2
	fi
done
duration=$SECONDS
PODTIMES+=("$duration")
echo "VPC-FILE-CSI-TEST: Statefulset's pods Running: $(($duration / 60)) minutes and $(($duration % 60)) seconds elapsed."
for pod in $podlist; do
	echo "VPC-FILE-CSI-TEST:               POD: $pod"
done
if [[ $it -ge $IT_COUNT ]]; then
	echo "[`date`] Wait Statefulset's pods Running TIMEOUT"
	EXIT_CODE=1
fi
echo "[`date`] Wait Statefulset's pods Running End"
if [[ -z "$podlist" ]]; then
	EXIT_CODE=1
fi
if [[ $EXIT_CODE -ne 0 ]]; then
	for pod in $podlist; do
		podname=${pod%%:*}
		kubectl describe pods -n $NAMESPACE $podname
		#kubectl exec -it -n $NAMESPACE $podname -- df -h
	done
	doCleanup $EXIT_CODE
	echo "VPC-FILE-CSI-TEST: Statefulset test: FAILED"
	exit $EXIT_CODE
else
	echo "VPC-FILE-CSI-TEST: Statefulset test: PASS"
fi
doCleanup $EXIT_CODE; EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]; then
	echo "VPC-FILE-CSI-TEST: Statefulset Cleanup: FAILED"
	kubectl logs -n kube-system -c iks-vpc-file-driver ibm-vpc-file-csi-controller-0 > ./blk-controller.log
	kubectl logs -n kube-system -c csi-attacher ibm-vpc-file-csi-controller-0 > ./blk-attacher.log
	node_servers=$(kubectl get pods -n kube-system -l "app=ibm-vpc-file-csi-driver" | grep 'csi-node'  | awk '{ print $1 }')
	for nodeSrvr in $node_servers; do
		kubectl logs -n kube-system -c iks-vpc-file-node-driver $nodeSrvr > ./blk-node-${nodeSrvr}.log
	done
fi

exit $EXIT_CODE
