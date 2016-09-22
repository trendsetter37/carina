package makecoe

import (
	"fmt"
	"net/http"

	"strings"

	"time"

	"github.com/getcarina/carina/common"
	"github.com/getcarina/libcarina"
	"github.com/pkg/errors"
)

// MakeCOE is an adapter between the cli and Carina (make-coe)
type MakeCOE struct {
	client  *libcarina.CarinaClient
	Account *Account
}

func (carina *MakeCOE) init() error {
	if carina.client == nil {
		carinaClient, err := carina.Account.Authenticate()
		if err != nil {
			return err
		}
		carina.client = carinaClient
	}
	return nil
}

// GetQuotas retrieves the quotas set for the account
func (carina *MakeCOE) GetQuotas() (common.Quotas, error) {
	return nil, errors.New("Not implemented")
}

// CreateCluster creates a new cluster and prints the cluster information
func (carina *MakeCOE) CreateCluster(name string, template string, nodes int) (common.Cluster, error) {
	err := carina.init()
	if err != nil {
		return nil, err
	}

	coe, hostType, err := getTemplateValues(template)
	if err != nil {
		return nil, err
	}

	common.Log.WriteDebug("[make-coe] Creating a %d-node %s cluster hosted on %s named %s", nodes, coe, hostType, name)
	createOpts := &libcarina.Cluster{
		Name:     name,
		COE:      coe,
		HostType: hostType,
		Nodes:    nodes,
	}

	result, err := carina.client.Create(createOpts)
	if err != nil {
		return nil, errors.Wrap(err, "[make-coe] Unable to create cluster")
	}

	cluster := &Cluster{Cluster: result}

	return cluster, nil
}

func getTemplateValues(template string) (coe string, hostType string, err error) {
	switch strings.ToLower(template) {
	case "kubernetes-dev":
		return "kubernetes", "vm", nil
	case "swarm-dev":
		return "swarm", "vm", nil
	default:
		return "", "", fmt.Errorf("Invalid template: %s", template)
	}
}

// GetClusterCredentials retrieves the TLS certificates and configuration scripts for a cluster by its id or name (if unique)
func (carina *MakeCOE) GetClusterCredentials(token string) (*libcarina.CredentialsBundle, error) {
	err := carina.init()
	if err != nil {
		return nil, err
	}

	common.Log.WriteDebug("[make-coe] Retrieving cluster credentials (%s)", token)
	creds, err := carina.client.GetCredentials(token)
	if err != nil {
		return nil, errors.Wrap(err, "[make-coe] Unable to retrieve the cluster credentials")
	}

	return creds, nil
}

// ListClusters prints out a list of the user's clusters to the console by its id or name (if unique)
func (carina *MakeCOE) ListClusters() ([]common.Cluster, error) {
	var clusters []common.Cluster

	err := carina.init()
	if err != nil {
		return clusters, err
	}

	common.Log.WriteDebug("[make-coe] Listing clusters")
	results, err := carina.client.List()
	if err != nil {
		return clusters, errors.Wrap(err, "[make-coe] Unable to list clusters")
	}

	for _, result := range results {
		cluster := &Cluster{Cluster: result}
		clusters = append(clusters, cluster)
	}

	return clusters, err
}

// RebuildCluster destroys and recreates the cluster by its id or name (if unique)
func (carina *MakeCOE) RebuildCluster(token string) (common.Cluster, error) {
	return nil, errors.New("Not implemented")
}

// GetCluster prints out a cluster's information to the console by its id or name (if unique)
func (carina *MakeCOE) GetCluster(token string) (common.Cluster, error) {
	err := carina.init()
	if err != nil {
		return nil, err
	}

	common.Log.WriteDebug("[make-coe] Retrieving cluster (%s)", token)
	result, err := carina.client.Get(token)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("[make-coe] Unable to retrieve cluster (%s)", token))
	}
	cluster := &Cluster{Cluster: result}

	return cluster, nil
}

// DeleteCluster permanently deletes a cluster by its id or name (if unique)
func (carina *MakeCOE) DeleteCluster(token string) (common.Cluster, error) {
	err := carina.init()
	if err != nil {
		return nil, err
	}

	common.Log.WriteDebug("[make-coe] Deleting cluster (%s)", token)
	result, err := carina.client.Delete(token)
	if err != nil {
		if httpErr, ok := err.(libcarina.HTTPErr); ok {
			if httpErr.StatusCode == http.StatusNotFound {
				common.Log.WriteWarning("Could not find the cluster (%s) to delete", token)
				cluster := newCluster()
				cluster.Status = "deleted"
				return cluster, nil
			}
		}

		return nil, errors.Wrap(err, fmt.Sprintf("[make-coe] Unable to delete cluster (%s)", token))
	}

	cluster := &Cluster{Cluster: result}

	return cluster, nil
}

// GrowCluster adds nodes to a cluster by its id or name (if unique)
func (carina *MakeCOE) GrowCluster(token string, nodes int) (common.Cluster, error) {
	return nil, errors.New("Not implemented")
}

// SetAutoScale is not supported
func (carina *MakeCOE) SetAutoScale(token string, value bool) (common.Cluster, error) {
	return nil, errors.New("make-coe does not support autoscaling")
}

// WaitUntilClusterIsActive waits until the prior cluster operation is completed
func (carina *MakeCOE) WaitUntilClusterIsActive(cluster common.Cluster) (common.Cluster, error) {
	return nil, errors.New("Not implemented")
}

// WaitUntilClusterIsDeleted polls the cluster status until either the cluster is gone or an error state is hit
func (carina *MakeCOE) WaitUntilClusterIsDeleted(cluster common.Cluster) error {
	isDone := func(cluster common.Cluster) bool {
		status := strings.ToUpper(cluster.GetStatus())
		return status == "deleted"
	}

	if isDone(cluster) {
		return nil
	}

	pollingInterval := 5 * time.Second
	for {
		cluster, err := carina.GetCluster(cluster.GetID())

		if err != nil {
			err = errors.Cause(err)

			// Gracefully handle a 404 Not Found when the cluster is deleted quickly
			if httpErr, ok := err.(libcarina.HTTPErr); ok {
				if httpErr.StatusCode == http.StatusNotFound {
					return nil
				}
			}

			return err
		}

		if isDone(cluster) {
			return nil
		}

		common.Log.WriteDebug("[make-coe] Waiting until cluster (%s) is deleted, currently in %s", cluster.GetName(), cluster.GetStatus())
		time.Sleep(pollingInterval)
	}
}
