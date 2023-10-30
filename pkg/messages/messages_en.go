/**
 * Copyright 2021 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package messages ...
package messages

import (
	"google.golang.org/grpc/codes"
)

// messagesEn ...
var messagesEn = map[string]Message{
	MethodUnimplemented: {
		Code:        MethodUnimplemented,
		Description: "'%s' CSI interface method not yet implemented",
		Type:        codes.Unimplemented,
		Action:      "The method you requested is not implemented.",
	},
	MethodUnsupported: {
		Code:        MethodUnsupported,
		Description: "'%s' CSI interface method is not supported",
		Type:        codes.Unimplemented,
		Action:      "The method you requested is not supported.",
	},
	MissingVolumeName: {
		Code:        MissingVolumeName,
		Description: "Volume name not provided",
		Type:        codes.InvalidArgument,
		Action:      "You must specify a volume name when creating a volume.",
	},
	MissingSnapshotName: {
		Code:        MissingSnapshotName,
		Description: "Snapshot name not provided",
		Type:        codes.InvalidArgument,
		Action:      "You must specify a snapshot name when creating a snapshot.",
	},
	MissingSourceVolumeID: {
		Code:        MissingSourceVolumeID,
		Description: "Volume ID not provided",
		Type:        codes.InvalidArgument,
		Action:      "You must a specify sourceVolumeID when creating a snapshot.",
	},
	UnsupportedVolumeContentSource: {
		Code:        UnsupportedVolumeContentSource,
		Description: "Invalid volumeContentSource.",
		Type:        codes.InvalidArgument,
		Action:      "You must specify a valid volumeContentSource when creating a snapshot.",
	},
	NoVolumeCapabilities: {
		Code:        NoVolumeCapabilities,
		Description: "Volume capabilities not provided",
		Type:        codes.InvalidArgument,
		Action:      "You must specify volume capabilities in the storage class when creating a volume.",
	},
	VolumeCapabilitiesNotSupported: {
		Code:        VolumeCapabilitiesNotSupported,
		Description: "Volume capabilities not supported",
		Type:        codes.InvalidArgument,
		Action:      "Verify the volume capabilities in your persistent volume claim are supported and formatted correctly",
	},
	InvalidParameters: {
		Code:        InvalidParameters,
		Description: "Failed to extract parameters",
		Type:        codes.InvalidArgument,
		Action:      "Verify your inputs and retry your request.",
	},
	ObjectNotFound: {
		Code:        ObjectNotFound,
		Description: "Object not found",
		Type:        codes.NotFound,
		Action:      "Review the 'BackendError' tag for more details.",
	},
	InternalError: {
		Code:        InternalError,
		Description: "Internal error occurred",
		Type:        codes.Internal,
		Action:      "Review the 'BackendError' tag for more details.",
	},
	VolumeAlreadyExists: {
		Code:        VolumeAlreadyExists,
		Description: "A volume with the name '%s' already exists",
		Type:        codes.AlreadyExists,
		Action:      "Specify a different volume name to create a new volume.",
	},
	SnapshotAlreadyExists: {
		Code:        SnapshotAlreadyExists,
		Description: "A snapshot with the name '%s' already exists for volume '%s'",
		Type:        codes.AlreadyExists,
		Action:      "Specify a different snapshot name to create a new snapshot.",
	},
	VolumeInvalidArguments: {
		Code:        VolumeInvalidArguments,
		Description: "Invalid arguments for create volume",
		Type:        codes.InvalidArgument,
		Action:      "Verify your inputs and retry your request.",
	},
	VolumeCreationFailed: {
		Code:        VolumeCreationFailed,
		Description: "Failed to create volume",
		Type:        codes.Internal,
		Action:      "Review the error which return in BackendError tag",
	},
	EmptyVolumeID: {
		Code:        EmptyVolumeID,
		Description: "No VolumeID specified",
		Type:        codes.InvalidArgument,
		Action:      " specify volume ID for attach/detach or delete it",
	},
	EmptySnapshotID: {
		Code:        EmptySnapshotID,
		Description: "SnapshotID must be provided",
		Type:        codes.InvalidArgument,
		Action:      " specify snapshot ID for deletion",
	},
	EmptyNodeID: {
		Code:        EmptyNodeID,
		Description: "NodeID is empty",
		Type:        codes.InvalidArgument,
		Action:      "Review your node labels by running the 'kubectl get nodes --show-labels' command",
	},
	EndpointNotReachable: {
		Code:        EndpointNotReachable,
		Description: "IAM TOKEN exchange request failed.",
		Type:        codes.Unavailable,
		Action:      "Verify that the 'iks_token_exchange_endpoint_private_url' is reachable from the cluster. You can find this URL by running the 'kubectl get secret storage-secret-storage -n kube-system' command.",
	},
	Timeout: {
		Code:        Timeout,
		Description: "IAM Token exchange endpoint is not reachable.",
		Type:        codes.DeadlineExceeded,
		Action:      "Wait a few minutes and try again. If the error persists user can open a container network issue.",
	},
	FailedPrecondition: {
		Code:        FailedPrecondition,
		Description: "Provider is not ready.",
		Type:        codes.FailedPrecondition,
		Action:      "Wait a few minutes and try again.",
	},
	NoStagingTargetPath: {
		Code:        NoStagingTargetPath,
		Description: "A staging target path was not specified",
		Type:        codes.InvalidArgument,
		Action:      " check if there is any error in POD describe related with volume attach",
	},
	NoTargetPath: {
		Code:        NoTargetPath,
		Description: "No target path specified",
		Type:        codes.InvalidArgument,
		Action:      " check if there is any error in POD describe related with volume attach",
	},
	MountPointValidateError: {
		Code:        MountPointValidateError,
		Description: "Failed to check whether target path '%s' is a mount point",
		Type:        codes.FailedPrecondition,
		Action:      "Review the pod details by running 'kubectl describe pod POD'.",
	},
	UnmountFailed: {
		Code:        UnmountFailed,
		Description: "Failed to unmount the '%s' target path",
		Type:        codes.Internal,
		Action:      "Review the pod details by running 'kubectl describe pod POD'.",
	},
	MountFailed: {
		Code:        MountFailed,
		Description: "Failed to mount '%q' at '%q'",
		Type:        codes.Internal,
		Action:      "Review the pod details by running 'kubectl describe pod POD'.",
	},
	EmptyDevicePath: {
		Code:        EmptyDevicePath,
		Description: "No staging device path specified",
		Type:        codes.InvalidArgument,
		Action:      "Specify a device path and retry your request.",
	},
	DevicePathFindFailed: {
		Code:        DevicePathFindFailed,
		Description: "Failed to find '%s' device path",
		Type:        codes.Internal,
		Action:      "Verify the device path and retry your request.",
	},
	DevicePathNotFound: {
		Code:        DevicePathNotFound,
		Description: "Device path '%s' is not present",
		Type:        codes.Internal,
		Action:      "List your volume attachments by using the 'ibmcloud ks storage attachments --worker WORKER-ID --cluster CLUSTER-ID | grep VOLUME-ID'. If the volume has an 'Attached' status, open a ticket and select 'VPC' for the problem type. Otherwise, open a ticket and select 'IBM Cloud Kubernetes Service' as the problem type.",
	},
	TargetPathCheckFailed: {
		Code:        TargetPathCheckFailed,
		Description: "Failed to check if staging target path '%s' exists",
		Type:        codes.Internal,
		Action:      " check if there is any error in POD describe related with volume attach",
	},
	TargetPathCreateFailed: {
		Code:        TargetPathCreateFailed,
		Description: "Failed to create target path '%s'",
		Type:        codes.Internal,
		Action:      " check if there is any error in POD describe related with volume attach",
	},
	VolumeMountCheckFailed: {
		Code:        VolumeMountCheckFailed,
		Description: "Failed to check if volume is already mounted on '%s'",
		Type:        codes.Internal,
		Action:      " check if there is any error in POD describe related with volume attach",
	},
	FormatAndMountFailed: {
		Code:        FormatAndMountFailed,
		Description: "Failed to format '%s' and mount it at '%s'",
		Type:        codes.Internal,
		Action:      " check if there is any error in POD describe related with volume attach",
	},
	NodeMetadataInitFailed: {
		Code:        NodeMetadataInitFailed,
		Description: "Failed to initialize node metadata",
		Type:        codes.NotFound, //i.e correct no need to change to other code
		Action:      " check the node labels as per BackendError, accordingly you may add the labels manually",
	},
	EmptyVolumePath: {
		Code:        EmptyVolumePath,
		Description: "Volume path can not be empty",
		Type:        codes.InvalidArgument,
		Action:      "Verify the volume path in your pod specification. Get the details of your pod by running the 'kubectl describe pod POD' command.",
	},
	DevicePathNotExists: {
		Code:        DevicePathNotExists,
		Description: "Device path '%s' does not exist for volume ID '%s'",
		Type:        codes.NotFound,
		Action:      "No device available at the specified path. Verify the device exists and the path is correct.",
	},
	BlockDeviceCheckFailed: {
		Code:        BlockDeviceCheckFailed,
		Description: "Failed to determine if volume '%s' is block device or not",
		Type:        codes.Internal,
		Action:      "Verify the volume details in your pod specification. Get the details of your pod by running the 'kubectl describe pod POD' command.",
	},
	GetDeviceInfoFailed: {
		Code:        GetDeviceInfoFailed,
		Description: "Failed to get device info",
		Type:        codes.Internal,
		Action:      "Verify the device information in your pod specification. Get the details of your pod by running the 'kubectl describe pod POD' command.",
	},
	GetFSInfoFailed: {
		Code:        GetFSInfoFailed,
		Description: "Failed to get FS info",
		Type:        codes.Internal,
		Action:      "Verify the file system details in your pod specification. Get the details of your pod by running the 'kubectl describe pod POD' command.",
	},
	DriverNotConfigured: {
		Code:        DriverNotConfigured,
		Description: "Driver name not configured",
		Type:        codes.Unavailable,
		Action:      "Specify the driver name and retry your request",
	},
	RemoveMountTargetFailed: {
		Code:        RemoveMountTargetFailed,
		Description: "Failed to remove '%q' mount target",
		Type:        codes.Internal,
		Action:      "Verify the mount targets in your pod specification. Get the details of your pod by running the 'kubectl describe pod POD' command.",
	},
	CreateMountTargetFailed: {
		Code:        CreateMountTargetFailed,
		Description: "Failed to create '%q' mount target",
		Type:        codes.Internal,
		Action:      "Verify the mount targets in your pod specification. Get the details of your pod by running the 'kubectl describe pod POD' command.",
	},
	ListVolumesFailed: {
		Code:        ListVolumesFailed,
		Description: "Failed to list volumes",
		Type:        codes.Internal,
		Action:      " check 'BackendError' tag for more details",
	},
	ListSnapshotsFailed: {
		Code:        ListSnapshotsFailed,
		Description: "Failed to list snapshots",
		Type:        codes.Internal,
		Action:      " check 'BackendError' tag for more details",
	},
	StartVolumeIDNotFound: {
		Code:        StartVolumeIDNotFound,
		Description: "The volume ID '%s' specified in the start parameter of the list volume call could not be found",
		Type:        codes.Aborted,
		Action:      "Verify that the startVolumeID is correct and you have access to the volume.",
	},
	StartSnapshotIDNotFound: {
		Code:        StartSnapshotIDNotFound,
		Description: "The snapshot ID '%s' specified in the start parameter of the list snapshot call could not be found",
		Type:        codes.Aborted,
		Action:      "Verify that the startSnapshotID is correct and you have access to the snapshot.",
	},
	FileSystemResizeFailed: {
		Code:        FileSystemResizeFailed,
		Description: "Failed to resize the file system",
		Type:        codes.Internal,
		Action:      "Review your PVC details by running the 'kubectl describe pvc PVC' command.",
	},
	VolumePathNotMounted: {
		Code:        VolumePathNotMounted,
		Description: "VolumePath '%s' is not mounted",
		Type:        codes.FailedPrecondition,
		Action:      "Verify the volume paths in your pod specification. Get the details of your pod by running the 'kubectl describe pod POD' command.",
	},
	SubnetIDListNotFound: {
		Code:        SubnetIDListNotFound,
		Description: "Cluster subnet list 'vpc_subnet_ids' is not defined",
		Type:        codes.FailedPrecondition,
		Action:      "Verify that the 'ibm-cloud-provider-data' configmap exists and the'vpc_subnet_ids' field contains valid subnets. Run the 'kubectl get configmap ibm-cloud-provider-data -n kube-system -o yaml' command and review the 'BackendError' tag for more details.",
	},
	SubnetFindFailed: {
		Code:        SubnetFindFailed,
		Description: "A subnet with the specified zone '%s' and available cluster subnet list '%s' could not be found.",
		Type:        codes.FailedPrecondition,
		Action:      "Verify that the 'vpc_subnet_ids' field contains valid subnetIds. Run the 'kubectl get configmap ibm-cloud-provider-data -n kube-system -o yaml' command and review the 'BackendError' tag for more details.",
	},
}

// InitMessages ...
func InitMessages() map[string]Message {
	return messagesEn
}
