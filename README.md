# ibm-csi-common
Common library for implementing ibm csi driver for kubernetes

## Clone GHE repository

- Make a go workspace
  - `mkdir ibm-csi-common-ws`

- Make the go folder structure and the initial src folders and clone the repository
  - `mkdir -p ibm-csi-common-ws/src/github.com/IBM`
  - `cd ibm-csi-common-ws/src/github.com/IBM/`
  - `git clone https://github.com/IBM/ibm-csi-common.git`
  - `cd ibm-csi-common`
  - `make deps`
  - `make test`
