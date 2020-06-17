.PHONY: deps-update \
		build-bin \
		unittests \
		verify


export DOCKERFILE?=Dockerfile
export IMAGE_BASE?=quay.io/fpaoline/network-metrics-daemon
export TAG?=latest
export NAMESPACE?=openshift-network-metrics
export MONITORING_NAMESPACE?=openshift-monitoring
export KUBE_EXEC?=oc
export KUBE_RBAC_PROXY?=quay.io/openshift/origin-kube-rbac-proxy:latest
export IMAGE_TAG:=$(IMAGE_BASE):$(TAG)



deps-update:
	go mod tidy && \
	go mod vendor

build-bin:
	go build --mod=vendor -ldflags "-X main.build=$$(git rev-parse HEAD)" -o bin/network-metrics-daemon
	chmod +x bin/network-metrics-daemon

unittests: verify
	go test ./...

image: ; $(info Building image...)
	docker build -f $(DOCKERFILE) -t $(IMAGE_TAG) .

image_push: ; $(info Building image...)
	docker image push $(IMAGE_TAG)

deploy:
	hack/deploy.sh

deploy-k8s:
	DEPLOYMENT_FLAVOUR="-k8s" hack/deploy.sh

get-tools:
	hack/get_tools.sh

verify: get-tools
	./hack/check_gofmt.sh
	./hack/check_golint.sh
	./hack/check_changes.sh
