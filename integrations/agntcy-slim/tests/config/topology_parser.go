package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

// Auth represents authentication configuration
type Auth struct {
	SpireJwt bool `yaml:"spireJwt,omitempty"`
}

// Client represents a client configuration in the topology
type Client struct {
	Auth        Auth     `yaml:"auth"`
	SpireMtls   bool     `yaml:"spireMtls,omitempty"`
	ConnectedTo []string `yaml:"connectedTo"`
	Image       string   `yaml:"image"`
	Cmd         string   `yaml:"cmd"`
	Args        []string `yaml:"args"`
	AssertFor   string   `yaml:"assertFor"`
}

// Server represents a server configuration in the topology
type Server struct {
	Auth              Auth `yaml:"auth"`
	SpireMtls         bool `yaml:"spireMtls,omitempty"`
	DeployAsDaemonSet bool `yaml:"deployAsDaemonSet,omitempty"`
	ReplicaCount      int  `yaml:"replicaCount,omitempty"`
}

// Topology represents the topology configuration
type Topology struct {
	Clients  map[string]Client `yaml:"clients"`
	Clusters map[string]Server `yaml:"clusters"`
}

// Config represents the root configuration structure
type Config struct {
	Topology Topology `yaml:"topology"`
}

// ParseTopology parses a YAML file and returns a Config struct
func ParseTopology(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	return ParseYAML(data)
}

// ParseYAML parses YAML data and returns a Config struct
func ParseYAML(data []byte) (*Config, error) {
	var config Config
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &config, nil
}

// ToYAML converts the Config struct back to YAML bytes
func (c *Config) ToYAML() ([]byte, error) {
	data, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config to YAML: %w", err)
	}
	return data, nil
}

// GetClient returns a client by name
func (c *Config) GetClient(name string) (Client, bool) {
	client, exists := c.Topology.Clients[name]
	return client, exists
}

// GetCluster returns a cluster by name
func (c *Config) GetCluster(name string) (Server, bool) {
	cluster, exists := c.Topology.Clusters[name]
	return cluster, exists
}

// ListClients returns a slice of all client names
func (c *Config) ListClients() []string {
	clients := make([]string, 0, len(c.Topology.Clients))
	for name := range c.Topology.Clients {
		clients = append(clients, name)
	}
	return clients
}

// ListClusters returns a slice of all cluster names
func (c *Config) ListClusters() []string {
	clusters := make([]string, 0, len(c.Topology.Clusters))
	for name := range c.Topology.Clusters {
		clusters = append(clusters, name)
	}
	return clusters
}
