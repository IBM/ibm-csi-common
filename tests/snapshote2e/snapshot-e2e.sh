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

error() {
     if [[ $? != 0 ]]; then
         echo "$1"
         exit
     fi
}

enableAddon() {
    ibmcloud ks cluster addon disable  vpc-block-csi-driver   -c $1
    sleep 30s
    
}

#Get cluster name
IN=$(kubectl config current-context)
error "Unable to fetch kubectl config"

IFS='/' read -ra CLUSTER_NAME <<< "$IN"
echo "Cluster Name - $CLUSTER_NAME"

#Fetch cluster addon version
addon_version=$(ibmcloud ks cluster addon ls -c $CLUSTER_NAME | grep vpc-block-csi-driver | awk '{print $2}')
error "Unable to fetch vpc-block-csi-driver addon version"
echo "$addon_version"

#Check if the addon version is 5.0
expected_version="5.0"
if [[ "$addon_version" == *"$expected_version"* ]]; then
    echo "Expected addon version is enabled"
else
    enableAddon $CLUSTER_NAME
fi





