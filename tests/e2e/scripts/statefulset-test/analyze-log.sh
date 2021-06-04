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
UNKOWNPARAM=()
while [[ $# -gt 0 ]]; do
        key="$1"

        case $key in
                -l|--log)
                LogFile="$2"
                shift
                shift
                ;;
                -p|--pvc)
                PvcName="$2"
                shift
                shift
                ;;
		-r|--run)
		run_count="$2"
		shift
		shift
		;;
		-o| --op)
		opertaion="$2"
		shift
		shift
		;;
		-h| --help)
		echo "Usage:$0 -l <log file> -p <pv name> -r <run count> -o <attach|detach>"
		exit 1
		shift
		shift
		;;
                *)
                UNKOWNPARAM+=("$1")
                shift
                ;;
        esac
done


LOGFILE=$LogFile
PVCNAME=$PvcName

if [[ ! -e $LOGFILE ]]; then
	echo "Log file not found: $LOGFILE"
	exit 1
fi

VOLUMEID=$(grep "Successfully created volume from VPC provider" $LOGFILE | grep $PVCNAME | jq '.VolumeDetails.id')
if [[ -z "$VOLUMEID" ]]; then
	echo "Invalid PVC: $PVCNAME"
	exit
fi

if [[ $opertaion == "attach" ]];then
	PUBLREQIDS=$(grep "ControllerPublishVolume" $LOGFILE  | grep $VOLUMEID | jq '.RequestID' | uniq )
	REQIDLIST=($PUBLREQIDS)
fi
if [[ $opertaion == "detach" ]];then
	UNPUBREQIDS=$(grep "ControllerUnpublishVolume" $LOGFILE  | grep $VOLUMEID  | jq '.RequestID' | uniq )
	REQIDLIST=($UNPUBREQIDS)
fi


if [[ $run_count -eq 1 ]]; then
	REQID=${REQIDLIST[0]}
else
	REQID=${REQIDLIST[1]}
fi
if [[ -z "$REQID" ]]; then
	echo "No request for PVC: PVCNAME"
	exit 1
fi

if [[ $opertaion == "attach" ]];then
	echo "PV Name   : [$PVCNAME]"
	echo "Attach Req: [$REQID]"
	grep $REQID $LOGFILE | grep -e 'ControllerPublishVolumeRequest' -e "Successfuly retrieved the volume attachment" -e '"Method":"POST"' -e '"Method":"GET"' -e 'Exit ControllerPublishVolume' | jq '.'
fi

if [[ $opertaion == "detach" ]];then
	echo "PV Name   : [$PVCNAME]"
	echo "Detach Req: [$REQID]"
	grep $REQID $LOGFILE | grep -e "In Controller Server's ControllerUnpublishVolume" -e "Successfully found volume attachment" -e '"Method":"DELETE"' -e '"Method":"GET"' -e 'Volume detachment is complete' | jq '.'
fi
if [[ $opertaion == "attach" ]];then
	StartTime=$(grep  $REQID $LOGFILE | grep "In Driver's ControllerPublishVolume" | jq '.ts')
	RiaaSStart=$(grep  $REQID $LOGFILE | grep '"Method":"POST"' | jq '.ts')
	EndTime=$(grep  $REQID $LOGFILE | grep "Exit ControllerPublishVolume()" | jq '.ts')
	StartTime=${StartTime//\"/}
	RiaaSStart=${RiaaSStart//\"/}
	EndTime=${EndTime//\"/}

	echo "PV Name       : [$PVCNAME]"
	echo "Attach Req. ID: [$REQID]"
	echo "Start Time    : $StartTime"
	echo "RiaaS Call    : $RiaaSStart"
	echo "End   Time    : $EndTime"
	start_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${StartTime%%.*} +%s)
	end_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${EndTime%%.*} +%s)
	echo "Driver $opertaion operation: $((( end_time - start_time )))sec"

	start_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${RiaaSStart%%.*} +%s)
	echo "RiaaS Vol $opertaion: $((( end_time - start_time )))sec"
fi

if [[ $opertaion == "detach" ]];then
	StartTime=$(grep  $REQID $LOGFILE | grep "In Controller Server's ControllerUnpublishVolume" | jq '.ts')
	RiaaSStart=$(grep  $REQID $LOGFILE | grep '"Method":"DELETE"' | jq '.ts')
	EndTime=$(grep  $REQID $LOGFILE | grep "Volume detachment is complete" | jq '.ts')
	StartTime=${StartTime//\"/}
	RiaaSStart=${RiaaSStart//\"/}
	EndTime=${EndTime//\"/}

	echo "PV Name       : [$PVCNAME]"
	echo "Detach Req. ID: [$REQID]"
	echo "Start Time    : $StartTime"
	echo "RiaaS Call    : $RiaaSStart"
	echo "End   Time    : $EndTime"
	start_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${StartTime%%.*} +%s)
	end_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${EndTime%%.*} +%s)
	echo "Driver $opertaion operation: $((( end_time - start_time )))sec"

	start_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${RiaaSStart%%.*} +%s)
	echo "RiaaS Vol $opertaion: $((( end_time - start_time )))sec"
fi
