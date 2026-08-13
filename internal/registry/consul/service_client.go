package consul

import (
	"context"
	"fmt"
	"time"

	consulapi "github.com/hashicorp/consul/api"
)

// consulAPIServiceClient implements ServiceClient using the official Consul API client.
// It queries healthy service instances and supports blocking queries for watch.
type consulAPIServiceClient struct {
	client      *consulapi.Client
	passingOnly bool
	watchTimeout time.Duration
}

// NewServiceClient creates a real Consul API service client.
//
// GetService queries healthy instances via Consul Health API. Watch uses
// Consul blocking queries (WaitIndex) so updates arrive near-real-time
// without polling.
func NewServiceClient(config *DiscoveryConfig) (ServiceClient, error) {
	apiConfig := consulapi.DefaultConfig()
	apiConfig.Address = config.Address
	apiConfig.Token = config.Token
	if config.Namespace != "" {
		apiConfig.Namespace = config.Namespace
	}
	if config.Partition != "" {
		apiConfig.Partition = config.Partition
	}

	client, err := consulapi.NewClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("consul: failed to create client: %w", err)
	}

	watchTimeout := config.WatchTimeout
	if watchTimeout == 0 {
		watchTimeout = 10 * time.Second
	}

	return &consulAPIServiceClient{
		client:       client,
		passingOnly:  true,
		watchTimeout: watchTimeout,
	}, nil
}

// GetService retrieves all healthy instances for a service.
func (c *consulAPIServiceClient) GetService(ctx context.Context, serviceName string) ([]*consulServiceInstance, error) {
	q := (&consulapi.QueryOptions{}).WithContext(ctx)
	entries, _, err := c.client.Health().Service(serviceName, "", c.passingOnly, q)
	if err != nil {
		return nil, fmt.Errorf("consul: failed to query service %s: %w", serviceName, err)
	}
	return toConsulServiceInstances(entries), nil
}

// Watch performs a blocking query and returns instances plus the new index.
// It blocks server-side until the service changes or watchTimeout elapses.
func (c *consulAPIServiceClient) Watch(ctx context.Context, serviceName string, lastIndex uint64) ([]*consulServiceInstance, uint64, error) {
	q := (&consulapi.QueryOptions{
		WaitIndex: lastIndex,
		WaitTime:  c.watchTimeout,
	}).WithContext(ctx)

	entries, meta, err := c.client.Health().Service(serviceName, "", c.passingOnly, q)
	if err != nil {
		return nil, 0, fmt.Errorf("consul: failed to watch service %s: %w", serviceName, err)
	}
	return toConsulServiceInstances(entries), meta.LastIndex, nil
}

// toConsulServiceInstances converts Consul service entries to the internal format.
func toConsulServiceInstances(entries []*consulapi.ServiceEntry) []*consulServiceInstance {
	instances := make([]*consulServiceInstance, 0, len(entries))
	for _, e := range entries {
		if e.Service == nil {
			continue
		}
		meta := e.Service.Meta
		if meta == nil {
			meta = make(map[string]string)
		}
		instances = append(instances, &consulServiceInstance{
			ID:      e.Service.ID,
			Name:    e.Service.Service,
			Address: e.Service.Address,
			Port:    e.Service.Port,
			Tags:    e.Service.Tags,
			Meta:    meta,
		})
	}
	return instances
}
