package create

import (
	"fmt"
	"strings"

	"github.com/kris-nova/logger"

	"github.com/weaveworks/eksctl/pkg/ami"
	api "github.com/weaveworks/eksctl/pkg/apis/eksctl.io/v1alpha4"
	"github.com/weaveworks/eksctl/pkg/eks"
)

// NewNodeGroupChecker validates a new nodegroup and applies defaults
func NewNodeGroupChecker(i int, ng *api.NodeGroup) error {
	if err := api.ValidateNodeGroup(i, ng); err != nil {
		return err
	}

	// apply defaults
	if ng.InstanceType == "" {
		ng.InstanceType = api.DefaultNodeType
	}
	if ng.AMIFamily == "" {
		ng.AMIFamily = ami.ImageFamilyAmazonLinux2
	}
	if ng.AMI == "" {
		ng.AMI = ami.ResolverStatic
	}

	if ng.SecurityGroups == nil {
		ng.SecurityGroups = &api.NodeGroupSGs{
			AttachIDs: []string{},
		}
	}
	if ng.SecurityGroups.WithLocal == nil {
		ng.SecurityGroups.WithLocal = api.NewBoolTrue()
	}
	if ng.SecurityGroups.WithShared == nil {
		ng.SecurityGroups.WithShared = api.NewBoolTrue()
	}

	if ng.AllowSSH {
		if ng.SSHPublicKeyPath == "" {
			ng.SSHPublicKeyPath = defaultSSHPublicKey
		}
	}

	if ng.VolumeSize > 0 {
		if ng.VolumeType == "" {
			ng.VolumeType = api.DefaultNodeVolumeType
		}
	}

	if ng.IAM == nil {
		ng.IAM = &api.NodeGroupIAM{}
	}
	if ng.IAM.WithAddonPolicies.ImageBuilder == nil {
		ng.IAM.WithAddonPolicies.ImageBuilder = api.NewBoolFalse()
	}
	if ng.IAM.WithAddonPolicies.AutoScaler == nil {
		ng.IAM.WithAddonPolicies.AutoScaler = api.NewBoolFalse()
	}
	if ng.IAM.WithAddonPolicies.ExternalDNS == nil {
		ng.IAM.WithAddonPolicies.ExternalDNS = api.NewBoolFalse()
	}

	return nil
}

// When passing the --without-nodegroup option, don't create nodegroups
func skipNodeGroupsIfRequested(cfg *api.ClusterConfig) {
	if withoutNodeGroup {
		cfg.NodeGroups = nil
		logger.Warning("cluster will be created without an initial nodegroup")
	}
}

func checkSubnetsGiven(cfg *api.ClusterConfig) bool {
	return cfg.VPC.Subnets != nil && len(cfg.VPC.Subnets.Private)+len(cfg.VPC.Subnets.Public) != 0
}

func checkSubnetsGivenAsFlags() bool {
	return len(*subnets[api.SubnetTopologyPrivate])+len(*subnets[api.SubnetTopologyPublic]) != 0
}

func checkVersion(ctl *eks.ClusterProvider, meta *api.ClusterMeta) error {
	switch meta.Version {
	case "auto":
		break
	case "":
		meta.Version = "auto"
	case "latest":
		meta.Version = api.LatestVersion
		logger.Info("will use version latest version (%s) for new nodegroup(s)", meta.Version)
	default:
		validVersion := false
		for _, v := range api.SupportedVersions() {
			if meta.Version == v {
				validVersion = true
			}
		}
		if !validVersion {
			return fmt.Errorf("invalid version %s, supported values: auto, latest, %s", meta.Version, strings.Join(api.SupportedVersions(), ", "))
		}
	}

	if v := ctl.ControlPlaneVersion(); v == "" {
		return fmt.Errorf("unable to get control plane version")
	} else if meta.Version == "auto" {
		meta.Version = v
		logger.Info("will use version %s for new nodegroup(s) based on control plane version", meta.Version)
	} else if meta.Version != v {
		hint := "--version=auto"
		if clusterConfigFile != "" {
			hint = "metadata.version: auto"
		}
		logger.Warning("will use version %s for new nodegroup(s), while control plane version is %s; to automatically inherit the version use %q", meta.Version, v, hint)
	}

	return nil
}
