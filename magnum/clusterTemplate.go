package magnum

import (
	"github.com/gophercloud/gophercloud/openstack/containerorchestration/v1/baymodels"
)

// ClusterTemplate represents a cluster template for make-coe
type ClusterTemplate struct {
	*baymodels.BayModel
}

// GetName returns the unique template name
func (template *ClusterTemplate) GetName() string {
	return template.Name
}

// GetCOE returns the container orchestration engine used by the cluster
func (template *ClusterTemplate) GetCOE() string {
	return template.COE
}

// GetHostType returns the underlying type of the host nodes, such as lxc or vm
func (template *ClusterTemplate) GetHostType() string {
	return template.ServerType
}
