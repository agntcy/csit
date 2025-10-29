package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/agntcy/csit/integrations/agntcy-slim/tests/config"
)

const (
	SlimMessagingPort        = "46357"
	ServerConfigTemplatePath = "config/server-config.tpl"
)

// SpireConfig represents the spire configuration section
type SpireConfig struct {
	Enabled bool `yaml:"enabled"`
}

// ServerConfigData represents the data structure for the server-config.tpl template
type ServerConfigData struct {
	Spire                  SpireConfig `yaml:"spire"`
	SlimPort               string      `yaml:"slimPort"`
	SlimControllerEndpoint string      `yaml:"slimControllerEndpoint"`
	ClusterName            string      `yaml:"clusterName"`
	ServiceName            string      `yaml:"serviceName"`
	DeployAsDaemonSet      bool        `yaml:"deployAsDaemonSet"`
	ReplicaCount           int         `yaml:"replicaCount"`
}

// GenerateServerConfig generates a server configuration file from the template
func GenerateConfigFromTemplate(templatePath, outputPath string, data ServerConfigData) error {
	// Parse the template file
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template file %s: %w", templatePath, err)
	}

	// Create the output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	// Execute the template with the provided data
	if err := tmpl.Execute(outputFile, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// GenerateClusterConfigs generates cluster configs for each cluster in the topology
func GenerateClusterConfigs(topology *config.Config, slimControllerEndpoint string, outputDir string) error {
	// Generate a config file for each cluster
	for clusterName, clusterConfig := range topology.Topology.Clusters {
		// Determine spire settings based on auth configuration
		spireEnabled := clusterConfig.SpireMtls || clusterConfig.Auth.SpireJwt

		deployAsDaemonSet := clusterConfig.DeployAsDaemonSet
		replicaCount := clusterConfig.ReplicaCount
		if replicaCount == 0 {
			replicaCount = 1
		}

		// Create template data
		data := ServerConfigData{
			Spire: SpireConfig{
				Enabled: spireEnabled,
			},
			SlimPort:               SlimMessagingPort,
			SlimControllerEndpoint: slimControllerEndpoint,
			ClusterName:            clusterName,
			ServiceName:            fmt.Sprintf("agntcy-%s-slim.%s.svc.cluster.local", clusterName, clusterName),
			DeployAsDaemonSet:      deployAsDaemonSet,
			ReplicaCount:           replicaCount,
		}

		// Generate server config file
		outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.yaml", clusterName))
		if err := GenerateConfigFromTemplate(ServerConfigTemplatePath, outputPath, data); err != nil {
			return fmt.Errorf("failed to generate config for server %s: %w", clusterName, err)
		}

		fmt.Printf("Generated config for cluster '%s' at: %s\n", clusterName, outputPath)
	}

	return nil
}

func main() {
	topologyConfig := os.Getenv("TOPOLOGY_CONFIG")
	if topologyConfig == "" {
		log.Fatal("TOPOLOGY_CONFIG environment variable is not set")
	}
	slimControllerEndpoint := os.Getenv("SLIM_CONTROLLER_ENDPOINT")
	if slimControllerEndpoint == "" {
		log.Fatal("SLIM_CONTROLLER_ENDPOINT environment variable is not set")
	}
	// Parse the fire-and-forget.yaml configuration
	topology, err := config.ParseTopology(topologyConfig)
	if err != nil {
		log.Fatalf("Failed to parse configuration: %v", err)
	}

	fmt.Println("Configuration loaded successfully!")
	fmt.Printf("Found %d clusters in topology\n", len(topology.Topology.Clusters))

	outputPath := "config/.gen"
	// Create the output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Generate cluster configs from topology
	err = GenerateClusterConfigs(topology, slimControllerEndpoint, outputPath)
	if err != nil {
		log.Fatalf("Failed to generate cluster configs: %v", err)
	}

	fmt.Println("All configurations generated successfully!")
}
