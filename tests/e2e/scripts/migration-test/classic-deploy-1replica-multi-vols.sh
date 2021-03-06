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
CORDONNODES=""

PVCCOUNT="single"
PVCSPEC="./tests/e2e/scripts/migration-test/specs/classic/ics-e2e-tester-single-pvcs.yaml"
DEPLOYSPEC="./tests/e2e/scripts/migration-test/specs/classic/ics-e2e-tester-single-deploy.yaml"

UNKOWNPARAM=()
while [[ $# -gt 0 ]]; do
        key="$1"
        case $key in
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

if [[ "$PVCCOUNT" != "single" ]]; then
	PVCSPEC="./tests/e2e/scripts/migration-test/specs/classic/ics-e2e-tester-multi-pvcs.yaml"
	DEPLOYSPEC="./tests/e2e/scripts/migration-test/specs/classic/ics-e2e-tester-multi-deploy.yaml"
fi

# ###############################
# Cleanup Function
# ###############################
function doCleanup {
	retcode=0

	echo "[`date`] Delete Deploy Start"
	kubectl delete -n $NAMESPACE -f $DEPLOYSPEC; rc=$?
	echo "[`date`] Wait Deploy Delete Start"
	while [[ $it -gt 0 && $it -le $IT_COUNT ]]; do
		active_pods=$(kubectl get pod -n $NAMESPACE -l "run=ics-e2e-tester"  -o name)
		if [[ -n "$active_pods" ]]; then
			echo "[`date`] Wait deploy delete... {${it}/${IT_COUNT}]"
			(( it += 1 ))
			sleep 3
		else
			it=-1
		fi
	done
	if [[ $it -ge $IT_COUNT ]]; then
		echo "[`date`] Wait Deploy Delete TIMEOUT"
		return 1
	fi
	echo "[`date`] Delete Deploy End"

	errc=0
	while [[ $errc -ge 0 && $errc -lt 10 ]]; do
		pvlist=$(kubectl get pvc -n $NAMESPACE -l "run=ics-e2e-tester" -o jsonpath="{range .items[*]}{.metadata.name}:{.spec.volumeName}{'\n'}"); rc=$?
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
	kubectl delete -n $NAMESPACE -f $PVCSPEC; rc=$?
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
				echo "[`date`] ${pvArray[$ind]}:${phase} {${it}/${IT_COUNT}]"
			else
				echo "[`date`] ${pvArray[$ind]}:ERROR    {${it}/${IT_COUNT}]"
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
	echo "CLASSIC-BLK-TEST: PV Deletion: PV Count: $pvCount Time: $(($duration / 60)) minutes and $(($duration % 60)) seconds elapsed."
	for pv in $pvlist; do
		echo "CLASSIC-BLK-TEST:              PV: $pv"
	done
	if [[ $it -ge $IT_COUNT ]]; then
		echo "[`date`] Wait PV Delete TIMEOUT"
		retcode=1
	fi
	echo "[`date`] Wait PV Delete End"

	echo "[`date`] Delete Namespace [$NAMESPACE]"
	kubectl delete namespace $NAMESPACE
	echo "[`date`] Deleted Namespace [$NAMESPACE]"

	return $retcode
}

# ###############################
# Main Function
# ###############################
echo "E2E Test: Deployment with one replica and multiple volumes"

seed=$(date +%s); tag=$(( $seed % 10000 ))
NAMESPACE=${NAMESPACE}-${tag}
echo "[`date`] Create Namespace [$NAMESPACE]"
kubectl create namespace $NAMESPACE; rc=$?
if [[ $rc -ne 0 ]]; then
	echo "CLASSIC-BLK-TEST: Deployment POD migration from Node-A to Node-B: FAILED"
	exit 1
fi
echo "[`date`] Created Namespace [$NAMESPACE]"

echo "[`date`] Create PVCs Start"
kubectl create -n $NAMESPACE -f $PVCSPEC; rc=$?
echo "[`date`] Create PVCs End"

echo "[`date`] Wait PVC Bound Start"
it=1
SECONDS=0
errc=0
while [[ $it -gt 0 && $it -le $IT_COUNT ]]; do
	success=0
        pvclist=$(2>&1 kubectl get pvc -n $NAMESPACE -l "run=ics-e2e-tester" -o jsonpath="{range .items[*]}{.metadata.namespace}:{.metadata.name}:{.status.phase}{'\n'}"); rc=$?
	if [[ $rc -ne 0 ]]; then
		if [[ "$pvclist" == *timeout* ]]; then
			echo "TIMEOUT: $pvclist"
			(( errc += 1 ))
			if [[ $errc -gt  10 ]]; then
				echo "CLASSIC-BLK-TEST: Deployment POD migration from Node-A to Node-B: FAILED"
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
echo "CLASSIC-BLK-TEST: PVC Creation: PVC Count: $pvcCount Time: $(($duration / 60)) minutes and $(($duration % 60)) seconds elapsed."
for pvc in $pvclist; do
	echo "CLASSIC-BLK-TEST:               PVC: $pvc"
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
	echo "[`date`] Delete PVCs Start"
	kubectl delete -n $NAMESPACE -f $PVCSPEC; rc=$?
	echo "[`date`] Delete PVCs End"
	echo "[`date`] Delete Namespace [$NAMESPACE]"
	kubectl delete namespace $NAMESPACE
	echo "[`date`] Deleted Namespace [$NAMESPACE]"

	echo "CLASSIC-BLK-TEST: Deployment POD migration from Node-A to Node-B: FAILED"
	exit $EXIT_CODE
fi

echo "[`date`] Create Deploy Start"
kubectl create -n $NAMESPACE -f $DEPLOYSPEC; rc=$?
echo "[`date`] Create Deploy End"

echo "[`date`] Wait Deployment POD Running Start"
it=1
SECONDS=0
errc=0
while [[ $it -gt 0 && $it -le $IT_COUNT ]]; do
	success=0
        podlist=$(2>&1 kubectl get pods -n $NAMESPACE -l "run=ics-e2e-tester" -o jsonpath="{range .items[*]}{.metadata.name}:{.spec.nodeName}:{.status.phase}{'\n'}"); rc=$?
	if [[ $rc -ne 0 ]]; then
		if [[ "$pvclist" == *timeout* ]]; then
			echo "TIMEOUT: $pvclist"
			(( errc += 1 ))
			if [[ $errc -gt 10 ]]; then
				echo "CLASSIC-BLK-TEST: Deployment POD migration from Node-A to Node-B: FAILED"
				exit 1
			fi
			sleep 3
			continue
		fi
	fi
	errc=0
	if [[ -z "$podlist" ]]; then
		echo "Error: Deployment POD not found!"
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
echo "CLASSIC-BLK-TEST: Deployment POD Running: $(($duration / 60)) minutes and $(($duration % 60)) seconds elapsed."
for pod in $podlist; do
	echo "CLASSIC-BLK-TEST:               POD: $pod"
done
if [[ $it -ge $IT_COUNT ]]; then
	echo "[`date`] Wait POD Running TIMEOUT"
	EXIT_CODE=1
fi
echo "[`date`] Wait POD Running End"
if [[ -z "$podlist" ]]; then
	EXIT_CODE=1
fi
if [[ $EXIT_CODE -ne 0 ]]; then
	for pod in $podlist; do
		podname=${pod%%:*}
		kubectl describe pods -n $NAMESPACE $podname
		kubectl exec -it -n $NAMESPACE $podname -- df -h
	done
	doCleanup
	echo "CLASSIC-BLK-TEST: Deployment POD migration from Node-A to Node-B: FAILED"
	exit $EXIT_CODE
fi

errc=0
while [[ $errc -ge 0 && $errc -lt 10 ]]; do
	podlist=$(2>&1 kubectl get pods -n $NAMESPACE -l "run=ics-e2e-tester" -o jsonpath="{range .items[*]}{.metadata.name}:{.spec.nodeName}{'\n'}"); rc=$?
	if [[ $rc -ne 0 ]]; then
		echo "Warning: $pvclist"
		(( errc += 1 ))
		if [[ "$podlist" == *timeout* ]]; then
			sleep 3
		fi
	else
		errc=-1
	fi
done
if [[ $errc -ge 0 ]]; then
	EXIT_CODE=1
fi
for pod in $podlist; do
	podname=${pod%%:*}
	nodename=${pod##*:}
	echo "[`date`] Examine POD [$pod] Start"
	kubectl describe pods -n $NAMESPACE $podname
	kubectl exec -it -n $NAMESPACE $podname -- df -h
	echo "[`date`] Examine POD [$pod] End"
done

if [[ $EXIT_CODE -ne 0 ]]; then
	doCleanup
	echo "CLASSIC-BLK-TEST: Deployment POD migration from Node-A to Node-B: FAILED"
	exit $EXIT_CODE
fi

for pod in $podlist; do
	podname=${pod%%:*}
	nodename=${pod##*:}
	if [[ -z "$CORDONNODES" ]]; then
		CORDONNODES="$nodename"
	else
		CORDONNODES="$CORDONNODES $nodename"
	fi
	echo "[`date`] Cordon Node [$nodename] Start"
	kubectl cordon $nodename
	echo "[`date`] Cordon Node [$nodename] End"
	echo "[`date`] Delete POD [$pod] Start"
	kubectl delete pods -n $NAMESPACE $podname
	echo "[`date`] Delete POD [$pod] End"
done

echo "[`date`] Wait Deployment POD Running Start"
it=1
SECONDS=0
errc=0
while [[ $it -gt 0 && $it -le $IT_COUNT ]]; do
	success=0
        podlist=$(2>&1 kubectl get pods -n $NAMESPACE -l "run=ics-e2e-tester" -o jsonpath="{range .items[*]}{.metadata.name}:{.spec.nodeName}:{.status.phase}{'\n'}"); rc=$?
	if [[ $rc -ne 0 ]]; then
		if [[ "$pvclist" == *timeout* ]]; then
			echo "TIMEOUT: $pvclist"
			(( errc += 1 ))
			if [[ $errc -gt 10 ]]; then
				echo "CLASSIC-BLK-TEST: Deployment POD migration from Node-A to Node-B: FAILED"
				exit 1
			fi
			sleep 3
			continue
		fi
	fi
	errc=0
	if [[ -z "$podlist" ]]; then
		echo "Error: Deployment POD not found!"
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
echo "CLASSIC-BLK-TEST: Deployment POD Running: $(($duration / 60)) minutes and $(($duration % 60)) seconds elapsed."
for pod in $podlist; do
	echo "CLASSIC-BLK-TEST:               POD: $pod"
done
if [[ $it -ge $IT_COUNT ]]; then
	echo "[`date`] Wait POD Running TIMEOUT"
	EXIT_CODE=1
fi
echo "[`date`] Wait POD Running End"
if [[ -z "$podlist" ]]; then
	EXIT_CODE=1
fi

podlist=$(kubectl get pods -n $NAMESPACE -l "run=ics-e2e-tester" -o jsonpath="{range .items[*]}{.metadata.name}:{.spec.nodeName}{'\n'}")
for pod in $podlist; do
	podname=${pod%%:*}
	kubectl describe pods -n $NAMESPACE $podname
	kubectl exec -it -n $NAMESPACE $podname -- df -h
done
for pod in $CORDONNODES; do
	echo "[`date`] UnCordon Node [$nodename] Start"
	kubectl uncordon $nodename
	echo "[`date`] UnCordon Node [$nodename] End"
done
if  [[ $EXIT_CODE -eq 0 ]]; then
	echo "CLASSIC-BLK-TEST: Deployment POD migration from Node-A to Node-B: PASSED"
else
	echo "CLASSIC-BLK-TEST: Deployment POD migration from Node-A to Node-B: FAILED"
fi
doCleanup; EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]; then
        echo "VPC-BLK-CSI-TEST: Deployment Cleanup: FAILED"
fi
exit $EXIT_CODE
