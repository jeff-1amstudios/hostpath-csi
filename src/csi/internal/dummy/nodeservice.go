package dummy

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type nodeService struct {
	d             *Driver
	caps          []*csi.NodeServiceCapability
	accessibility *csi.Topology

	pendingVolOpts sync.Map
}

func newNodeService(d *Driver) csi.NodeServer {
	supportedRpcs := []csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
	}

	var caps []*csi.NodeServiceCapability
	for _, c := range supportedRpcs {
		caps = append(caps, &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: c,
				},
			},
		})
	}

	return &nodeService{
		d:    d,
		caps: caps,
	}
}

func (ns *nodeService) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: ns.caps,
	}, nil
}

func (ns *nodeService) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId: ns.d.NodeID,
	}, nil
}

func (ns *nodeService) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if err := validateNodePublishVolumeRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var (
		volID      = req.GetVolumeId()
		targetPath = req.GetTargetPath()
	)

	if _, isPending := ns.pendingVolOpts.LoadOrStore(volID, true); isPending {
		// CO should try again
		return nil, status.Error(codes.Aborted, fmt.Sprintf("volume %s is already being processed", volID))
	}
	defer ns.pendingVolOpts.Delete(volID)

	// from stagevolume
	stagingTargetPath := "/var/lib/csi-data/" + volID
	log.Printf("hello world!!\n")
	log.Printf("staging target path is %s", stagingTargetPath)

	err := os.Mkdir(stagingTargetPath, 0777)
	if err != nil {
		log.Print("Failed to mkdir stagingpath:", err)
	}
	// if mounted, err := isMountpoint(stagingTargetPath); err != nil {
	// 	return nil, status.Error(codes.Internal, err.Error())
	// } else if mounted {
	// 	// Already mounted
	// 	return &csi.NodePublishVolumeResponse{}, nil
	// }
	err = os.Mkdir(targetPath, 0777)
	if err != nil {
		log.Print("Failed to mkdir targetpath:", err)
	}

	if err := (bindMounter{}).mount(stagingTargetPath, targetPath); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	//cacheStageMount(volID, stagingTargetPath, ns.d.DriverOpts.MountCachePath)
	//cachePublishMount(volID, stagingTargetPath, targetPath, ns.d.DriverOpts.MountCachePath)

	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *nodeService) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if err := validateNodeUnpublishVolumeRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	log.Print("NodeUnpublishVolume")

	var (
		volID      = req.GetVolumeId()
		targetPath = req.GetTargetPath()
	)

	if err := (bindMounter{}).unmount(targetPath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmount bind %s for volume %s: %v", targetPath, volID, err))
	}

	stagingTargetPath := "/var/lib/csi-data/" + volID

	if err := rmMountpoint(targetPath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to remove mountpoint %s for volume %s: %v", targetPath, volID, err))
	}

	if err := rmMountpoint(stagingTargetPath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to remove staging target path %s for volume %s: %v", targetPath, volID, err))
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeService) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	if err := validateNodeStageVolumeRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var (
		volID             = req.GetVolumeId()
		stagingTargetPath = req.GetStagingTargetPath()
	)

	if _, isPending := ns.pendingVolOpts.LoadOrStore(volID, true); isPending {
		// CO should try again
		return nil, status.Error(codes.Aborted, fmt.Sprintf("volume %s is already being processed", volID))
	}
	defer ns.pendingVolOpts.Delete(volID)

	if mounted, err := isMountpoint(stagingTargetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	} else if mounted {
		// Already mounted
		return &csi.NodeStageVolumeResponse{}, nil
	}

	if err := (fuseMounter{}).mount("", stagingTargetPath); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	cacheStageMount(volID, req.GetStagingTargetPath(), ns.d.DriverOpts.MountCachePath)

	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *nodeService) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	if err := validateNodeUnstageVolumeRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var (
		volID             = req.GetVolumeId()
		stagingTargetPath = req.GetStagingTargetPath()
	)

	if _, isPending := ns.pendingVolOpts.LoadOrStore(volID, true); isPending {
		// CO should try again
		return nil, status.Error(codes.Aborted, fmt.Sprintf("volume %s is already being processed", volID))
	}
	defer ns.pendingVolOpts.Delete(volID)

	if err := (fuseMounter{}).unmount(stagingTargetPath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmount %s for volume %s: %v", stagingTargetPath, volID, err))
	}

	forgetStageMount(volID, ns.d.DriverOpts.MountCachePath)

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *nodeService) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "RPC not implemented")
}

func (ns *nodeService) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "RPC not implemented")
}
