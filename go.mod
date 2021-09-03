module github.com/IBM/ibm-csi-common

go 1.16

require (
	github.com/IBM/ibmcloud-volume-interface v1.0.0-beta7
	github.com/IBM/ibmcloud-volume-vpc v1.0.0-beta9.0.20210903054441-2b1749b4a013
	github.com/container-storage-interface/spec v1.2.0
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20191011121108-aa519ddbe484 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.3.1 // indirect
	github.com/golang/protobuf v1.4.2
	github.com/google/btree v1.0.1 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/googleapis/gnostic v0.0.0-20170729233727-0c5108395e2d // indirect
	github.com/gregjones/httpcache v0.0.0-20170728041850-787624de3eb7 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/onsi/ginkgo v1.10.3
	github.com/onsi/gomega v1.7.1
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pierrre/gotestcover v0.0.0-20160517101806-924dca7d15f0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.7.1
	github.com/satori/go.uuid v1.2.0
	github.com/spf13/cobra v0.0.5 // indirect
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.15.0
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba // indirect
	google.golang.org/appengine v1.5.0 // indirect
	google.golang.org/genproto v0.0.0-20191114150713-6bbd007550de // indirect
	google.golang.org/grpc v1.27.0
	google.golang.org/protobuf v1.23.0
	gopkg.in/inf.v0 v0.9.0 // indirect
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	k8s.io/api v0.0.0-20190614205929-e4e27c96b39a
	k8s.io/apiextensions-apiserver v0.0.0-20190315093550-53c4693659ed // indirect
	k8s.io/apimachinery v0.0.0-20190612125636-6a5db36e93ad
	k8s.io/apiserver v0.0.0-20190614170213-5c4672425150 // indirect
	k8s.io/cli-runtime v0.0.0-20190314001948-2899ed30580f // indirect
	k8s.io/client-go v11.0.1-0.20190516230509-ae8359b20417+incompatible
	k8s.io/cluster-bootstrap v0.0.0-20190612131323-aa3fd9f69a09 // indirect
	k8s.io/component-base v0.0.0-20190617074208-2b0aae80ca81 // indirect
	k8s.io/csi-translation-lib v0.0.0-20190615091142-9ff632302e7e // indirect
	k8s.io/kube-aggregator v0.0.0-20190314000639-da8327669ac5 // indirect
	k8s.io/kube-openapi v0.0.0-20190603182131-db7b694dc208 // indirect
	k8s.io/kubernetes v1.14.2
	k8s.io/utils v0.0.0-20190607212802-c55fbcfc754a // indirect
	sigs.k8s.io/kustomize v2.0.3+incompatible // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190516230258-a675ac48af67
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190313205120-d7deff9243b1
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190313205120-8b27c41bdbb1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190516232619-2bf8e45c8454
)
