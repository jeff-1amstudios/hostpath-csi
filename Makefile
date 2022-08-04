BUILD_DIR ?= build
IMAGE ?= dummy-fuse-csi
IMAGE_TAG ?= local
VERSION ?= $(shell git describe --long --tags --dirty --always)
CSI_GOLDFLAGS := "-w -s -X 'dummy-fuse-csi/internal/dummy/version.Version=${VERSION}'"
WORKLOAD_GOLDFLAGS := "-w -s"

$(shell mkdir -p $(BUILD_DIR))

all: dummy-fuse-csi

dummy-fuse-csi:
	cd src/csi; CGO_ENABLED=0 go build -ldflags $(CSI_GOLDFLAGS) -o ../../$(BUILD_DIR)/$@ cmd/main.go

image: dummy-fuse-csi
	docker build -f ./Dockerfile $(BUILD_DIR) -t $(IMAGE):$(IMAGE_TAG)

clean:
	rm -rf $(BUILD_DIR)

.PHONY: all clean dummy-fuse-csi
