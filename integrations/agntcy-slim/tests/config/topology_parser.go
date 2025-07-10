package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

// Auth represents authentication configuration
type Auth struct {
	SpireJwt  bool `yaml:"spireJwt,omitempty"`
	SpireMtls bool `yaml:"spireMtls,omitempty"`
}

// Client represents a client configuration in the topology
type Client struct {
	Auth        Auth     `yaml:"auth"`
	ConnectedTo []string `yaml:"connectedTo"`
	Image       string   `yaml:"image"`
	Cmd         string   `yaml:"cmd"`
	Args        []string `yaml:"args"`
}

// Server represents a server configuration in the topology
type Server struct {
	Auth   Auth     `yaml:"auth"`
	Routes []string `yaml:"routes"`
}

// Topology represents the topology configuration
type Topology struct {
	Clients map[string]Client `yaml:"clients"`
	Servers map[string]Server `yaml:"servers"`
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

// GetServer returns a server by name
func (c *Config) GetServer(name string) (Server, bool) {
	server, exists := c.Topology.Servers[name]
	return server, exists
}

// ListClients returns a slice of all client names
func (c *Config) ListClients() []string {
	clients := make([]string, 0, len(c.Topology.Clients))
	for name := range c.Topology.Clients {
		clients = append(clients, name)
	}
	return clients
}

// ListServers returns a slice of all server names
func (c *Config) ListServers() []string {
	servers := make([]string, 0, len(c.Topology.Servers))
	for name := range c.Topology.Servers {
		servers = append(servers, name)
	}
	return servers
}
