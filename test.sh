#!/bin/bash
SNAP_ADDON_VERSION=5.0.0
CLUSTER_ADDON_VER=5.0.1 
compare=`echo | awk "{ print ($CLUSTER_ADDON_VER >= $SNAP_ADDON_VERSION)?1 : 0 }"`
 echo $compare
 if [[ $compare -eq 1 ]]; then
	echo "$CLUSTER_ADDON_VER is bigger"
else
	echo "$SNAP_ADDON_VERSION is bigger"
fi

