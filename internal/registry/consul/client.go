package consul

import (
	"context"
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
)

// consulAPIClient implements Client using the official Consul API client.
// It registers services with the local Consul agent via HTTP.
type consulAPIClient struct {
	client *consulapi.Client
}

// NewClient creates a real Consul API client backed by the Consul agent HTTP API.
//
// The returned Client performs actual registration against the Consul agent
// at config.Address. Tests can inject a mock via WithClient instead.
func NewClient(config *RegistrarConfig) (Client, error) {
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

	return &consulAPIClient{client: client}, nil
}

// Register registers the service with the Consul agent.
// The context is not used for cancellation because the Consul agent API
// is a synchronous HTTP call; timeouts are controlled by the agent client.
func (c *consulAPIClient) Register(ctx context.Context, instance *consulServiceInstance) error {
	reg := &consulapi.AgentServiceRegistration{
		ID:      instance.ID,
		Name:    instance.Name,
		Address: instance.Address,
		Port:    instance.Port,
		Tags:    instance.Tags,
		Meta:    instance.Meta,
	}

	// Attach health check configuration if present.
	if instance.Check != nil || len(instance.Checks) > 0 {
		check := c.toAgentServiceCheck(instance.Check)
		if check != nil {
			reg.Check = check
		}
	}

	if err := c.client.Agent().ServiceRegister(reg); err != nil {
		return fmt.Errorf("consul: failed to register service %s: %w", instance.Name, err)
	}
	return nil
}

// Deregister removes the service from the Consul agent.
func (c *consulAPIClient) Deregister(ctx context.Context, serviceID string) error {
	if err := c.client.Agent().ServiceDeregister(serviceID); err != nil {
		return fmt.Errorf("consul: failed to deregister service %s: %w", serviceID, err)
	}
	return nil
}

// toAgentServiceCheck converts the internal check config to the Consul API format.
func (c *consulAPIClient) toAgentServiceCheck(check *consulCheck) *consulapi.AgentServiceCheck {
	if check == nil {
		return nil
	}

	agentCheck := &consulapi.AgentServiceCheck{
		CheckID: check.CheckID,
		Name:    check.Name,
	}

	switch {
	case check.HTTP != "":
		agentCheck.HTTP = check.HTTP
	case check.TCP != "":
		agentCheck.TCP = check.TCP
	case check.TTL > 0:
		agentCheck.TTL = check.TTL.String()
	}

	if check.Interval > 0 {
		agentCheck.Interval = check.Interval.String()
	}
	if check.Timeout > 0 {
		agentCheck.Timeout = check.Timeout.String()
	}
	if check.DeregisterCriticalServiceAfter > 0 {
		agentCheck.DeregisterCriticalServiceAfter = check.DeregisterCriticalServiceAfter.String()
	}

	return agentCheck
}
