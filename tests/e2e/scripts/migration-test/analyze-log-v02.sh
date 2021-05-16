#!/bin/bash

#/******************************************************************************
# Copyright 2020 IBM Corp.
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
debug=0
script=0
UNKOWNPARAM=()
while [[ $# -gt 0 ]]; do
        key="$1"

        case $key in
		-v)
		debug=1
                shift
                ;;
		-vv)
		debug=2
                shift
                ;;
		-s|--script)
		script=1
                shift
                ;;
                -l|--log)
                LogFile="$2"
                shift
                shift
                ;;
                -p|--pvc)
                PvcNames="$2"
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
		echo "Usage:$0 [-d] -l <log file> -p <pv name> -r <run count> -o <attach|detach>"
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
PVCNAMELIST=${PvcNames//,/ }
REQIDLIST=()
RIASCALLTIMELIST=()
OPS_START=""
OPS_END=""
RiaaSTotalTime=0
os_name=$(uname)
#if [[ $script -eq 1 ]]; then
#    debug=0
#fi

if [[ ! -e $LOGFILE ]]; then
	echo "Log file not found: $LOGFILE"
	exit 1
fi

if [[ $opertaion == "detach" ]];then
    echo "**************** DETACH ****************"
else
    echo "**************** ATTACH ****************"
fi
for pvc_name in $PVCNAMELIST; do
    REQIDLIST=()
    volume_id=$(grep "Successfully created volume from VPC provider" $LOGFILE | grep $pvc_name | jq '.VolumeDetails.id')
    volume_id=${volume_id//\"/}
    if [[ -z "$volume_id" ]]; then
	echo "Invalid PVC: $pvc_name"
	exit
    fi
    if [[ $debug -eq 1 ]]; then
	echo "----------------------------------------"
	echo "PVC Name: [$pvc_name]"
	echo "Vol ID  : [$volume_id]"
    fi

    node_ids=""
    if [[ $opertaion == "attach" ]];then
	vol_publ_req_ids=$(grep "In Driver's ControllerPublishVolume" $LOGFILE  | grep $volume_id | jq -j '.ControllerPublishVolumeRequest.node_id,":",.RequestID,"\n"')
	node_ids=$(echo "$vol_publ_req_ids" | awk -F ":"  '{print $1}' | uniq)
	for nodeid in $node_ids; do
	    req_ids=$(echo "$vol_publ_req_ids" | grep $nodeid | awk -F ":" 'BEGIN { ORS="," }; { print $2 }')
	    req_ids=${req_ids%*,}
	    REQIDLIST+=("$req_ids")
	    if [[ $debug -eq 1 ]]; then
		echo ".........Req ID: [$req_ids]"
		echo "........Node ID: [$nodeid]"
	    fi
	done
	#PUBLREQIDS=$(grep "ControllerPublishVolume" $LOGFILE  | grep $volume_id | jq '.RequestID' | uniq )
	#REQIDLIST=($PUBLREQIDS)
    fi
    if [[ $opertaion == "detach" ]];then
	vol_unpubl_req_ids=$(grep "ControllerUnpublishVolume" $LOGFILE  | grep $volume_id  | jq '.RequestID' | uniq )
        vol_unpubl_req_ids=${vol_unpubl_req_ids//\"/}
	REQIDLIST=($vol_unpubl_req_ids)
    fi


    if [[ $run_count -eq 1 ]]; then
	REQ_ID=${REQIDLIST[0]}
    elif [[ $run_count -eq 2 ]]; then
	REQ_ID=${REQIDLIST[1]}
    else
	REQ_ID=${REQIDLIST[2]}
    fi
    REQ_ID=${REQ_ID//,/ }

    if [[ -z "$REQ_ID" ]]; then
	echo "No request for PVC: $pvc_name"
	exit 1
    fi

    if [[ $debug -ne 0 && $opertaion == "attach" ]];then
	for reqId in ${REQ_ID}; do
		nodeid=$(grep "In Driver's ControllerPublishVolume" $LOGFILE  | grep $reqId | jq -j '.ControllerPublishVolumeRequest.node_id')
		echo ".....Attach Req: [$reqId]"
		echo "........Node-ID: [$nodeid]"
		if [[ $debug -eq 2 ]]; then
		   grep $reqId $LOGFILE | grep -e 'ControllerPublishVolumeRequest' -e "Successfuly retrieved the volume attachment" -e '"Method":"POST"' -e '"Method":"GET"' -e 'Exit ControllerPublishVolume' | jq '.'
		fi
	done
    fi

    if [[ $debug -ne 0 && $opertaion == "detach" ]];then
	nodeid=$(grep $REQ_ID $LOGFILE |  grep "In Controller Server's ControllerUnpublishVolume" | jq -j '.Request.node_id')
	echo ".....Detach Req: [$REQ_ID]"
	echo "........Node ID: [$nodeid]"
	if [[ $debug -eq 2 ]]; then
	   grep $REQ_ID $LOGFILE | grep -e "In Controller Server's ControllerUnpublishVolume" -e "Successfully found volume attachment" -e '"Method":"DELETE"' -e '"Method":"GET"' -e 'Volume detachment is complete' | jq '.'
	fi
    fi

    StartTime=""
    RiaaSStart=""
    EndTime=""
    if [[ $opertaion == "attach" ]];then
	if [[ $debug -eq 1 ]]; then
	   echo "----------------------------------------"
	fi
	for reqId in ${REQ_ID}; do
	    start_time=$(grep $reqId $LOGFILE | grep "In Driver's ControllerPublishVolume" | jq '.ts')
	    if [[ -z "$StartTime" ]]; then
	       # StartTime=$(grep $reqId $LOGFILE | grep "In Driver's ControllerPublishVolume" | jq '.ts')
	       StartTime=${start_time}
	    fi
	    riaas_start=$(grep $reqId $LOGFILE | grep '"Method":"POST"' | jq '.ts')
	    end_time=$(grep $reqId $LOGFILE | grep "Exit ControllerPublishVolume()" | jq '.ts')
	    if [[ -z "$RiaaSStart" ]]; then # One of the req could be DUP
	       # RiaaSStart=$(grep $reqId $LOGFILE | grep '"Method":"POST"' | jq '.ts')
	       # EndTime=$(grep $reqId $LOGFILE | grep "Exit ControllerPublishVolume()" | jq '.ts')
	       RiaaSStart=${riaas_start}
	       EndTime=${end_time}
	    fi
	    if [[ $debug -eq 1 ]]; then
	    	echo "Attach Req. ID: [$reqId]"
	    	echo "    Start Time: [$start_time]"
	    	echo "    RiaaS Time: [$riaas_start]"
	    	echo "    End   Time: [$end_time]"
	    fi
        done
	if [[ -z "$RiaaSStart" ]]; then
		# RiaaSStart=$(grep  $REQ_ID $LOGFILE | grep '"Volume is already attached"' | jq '.ts')
		RiaaSStart=$EndTime
	fi
	StartTime=${StartTime//\"/}
	RiaaSStart=${RiaaSStart//\"/}
	EndTime=${EndTime//\"/}
	if [[ $debug -eq 1 ]]; then
	    echo "----------------------------------------"
	    echo "PV Name         : [$pvc_name]"
	    echo "Vol ID          : [$volume_id]"
	    echo "Attach Req. ID  : [$REQ_ID]"
	    echo "Attach Start    : $StartTime"
	    echo "RiaaS Call      : $RiaaSStart"
	    echo "Attach End      : $EndTime"
	fi

	if [[ "$os_name" == "Linux" ]]; then
	    start_time=$(date -d ${RiaaSStart} +%s)
	    end_time=$(date -d ${EndTime} +%s)
	else
	    start_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${RiaaSStart%%.*} +%s)
	    end_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${EndTime%%.*} +%s)
        fi
        timediff=$((( end_time - start_time )))
	RIASCALLTIMELIST+=("$timediff")
	RiaaSTotalTime=$((( $RiaaSTotalTime + timediff )))
	if [[ $debug -eq 1 ]]; then
	    echo "RiaaS call time : ${timediff}sec"
	fi

	#start_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${StartTime%%.*} +%s)
	#end_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${EndTime%%.*} +%s)
        #timediff=$((( end_time - start_time )))
	#echo "Driver $opertaion operation: ${timediff}sec"
    fi

    if [[ $opertaion == "detach" ]];then
	StartTime=$(grep  $REQ_ID $LOGFILE | grep "In Controller Server's ControllerUnpublishVolume" | jq '.ts')
	RiaaSStart=$(grep  $REQ_ID $LOGFILE | grep '"Method":"DELETE"' | jq '.ts')
	EndTime=$(grep  $REQ_ID $LOGFILE | grep "Volume detachment is complete" | jq '.ts')
	if [[ -z "$RiaaSStart" ]]; then
		RiaaSStart=$EndTime
	fi
	StartTime=${StartTime//\"/}
	RiaaSStart=${RiaaSStart//\"/}
	EndTime=${EndTime//\"/}

	if [[ $debug -eq 1 ]]; then
	    echo "----------------------------------------"
	    echo "PV Name         : [$pvc_name]"
	    echo "Vol ID          : [$volume_id]"
	    echo "Detach Req. ID  : [$REQ_ID]"
	    echo "Detach Start    : $StartTime"
	    echo "RiaaS Call      : $RiaaSStart"
	    echo "Detach End      : $EndTime"
	fi

	if [[ "$os_name" == "Linux" ]]; then
	    start_time=$(date -d ${RiaaSStart} +%s)
	    end_time=$(date -d ${EndTime} +%s)
	else
	    start_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${RiaaSStart%%.*} +%s)
	    end_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${EndTime%%.*} +%s)
        fi
        timediff=$((( end_time - start_time )))
	RIASCALLTIMELIST+=("$timediff")
	RiaaSTotalTime=$((( $RiaaSTotalTime + timediff )))
	if [[ $debug -eq 1 ]]; then
	    echo "RiaaS call time : ${timediff}sec"
	fi

	##start_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${StartTime%%.*} +%s)
	##end_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${EndTime%%.*} +%s)
        ##timediff=$((( end_time - start_time )))
	##echo "Driver $opertaion operation: ${timediff}sec"
    fi
    if [[ -z "$OPS_START" ]]; then
       OPS_START="$StartTime"
       OPS_END="$EndTime"
    else
	if [[ "$os_name" == "Linux" ]]; then
	    time01=$(date -d ${OPS_START} +%s)
	    time02=$(date -d ${StartTime} +%s)
	else
            time01=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${OPS_START%%.*} +%s)
            time02=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${StartTime%%.*} +%s)
        fi
	timediff=$((( time01 - time02 )))
	if [[ $timediff -gt 0 ]]; then
		OPS_START=$StartTime
	fi
	if [[ "$os_name" == "Linux" ]]; then
	    time01=$(date -d ${OPS_END} +%s)
	    time02=$(date -d ${EndTime} +%s)
	else
            time01=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${OPS_END%%.*} +%s)
            time02=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${EndTime%%.*} +%s)
        fi
	timediff=$((( time01 - time02 )))
	if [[ ! $timediff -ge 0 ]]; then
		OPS_END=$EndTime
	fi

    fi
done
if [[ "$os_name" == "Linux" ]]; then
    start_time=$(date -d ${OPS_START} +%s)
    end_time=$(date -d ${OPS_END} +%s)
else
    start_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${OPS_START%%.*} +%s)
    end_time=$(date -j -f "%Y-%m-%dT%H:%M:%S" ${OPS_END%%.*} +%s)
fi
timediff=$((( end_time - start_time )))
ratio=$((( 10000 * RiaaSTotalTime / timediff )))
major=$((( ratio / 100 )))
minor=$((( ratio % 100 )))

echo -e "\n----------------------------------------"
echo "Driver Time: ${timediff}sec"
echo "RiaaS  Time: ${RiaaSTotalTime}sec (${major}.${minor}%)"
echo "----------------------------------------"
if [[ $script -eq 1 ]]; then
    echo "PERF_Driver_Time=${timediff}"
    echo "PERF_RiaaS_Time=${RiaaSTotalTime}"
    summary=$(printf " %ssec " "${RIASCALLTIMELIST[@]}")
    echo "PERF_Riaas_Time_Summary=[$summary]"
fi
echo "****************************************"
