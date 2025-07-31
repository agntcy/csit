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
	SlimMessagingPort            = "46357"
	ServerConfigTemplatePath     = "config/server-config.tpl"
	ServerConnConfigTemplatePath = "config/server-conn-config.tpl"
)

// SpireConfig represents the spire configuration section
type SpireConfig struct {
	Enabled bool `yaml:"enabled"`
}

// ServerConfigData represents the data structure for the server-config.tpl template
type ServerConfigData struct {
	Spire                  SpireConfig `yaml:"spire"`
	SlimHost               string      `yaml:"slimHost"`
	SlimPort               string      `yaml:"slimPort"`
	SlimControllerEndpoint string      `yaml:"slimControllerEndpoint"`
	ServiceName            string      `yaml:"serviceName"`
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

// GenerateServerConfigs generates server configs for each server in the topology
func GenerateServerConfigs(topology *config.Config, slimControllerEndpoint string, outputDir string) error {
	// Generate a config file for each server
	for serverName, serverConfig := range topology.Topology.Servers {
		// Determine spire settings based on auth configuration
		spireEnabled := serverConfig.SpireMtls || serverConfig.Auth.SpireJwt

		// Create template data
		data := ServerConfigData{
			Spire: SpireConfig{
				Enabled: spireEnabled,
			},
			SlimHost:               fmt.Sprintf("agntcy-%s", serverName),
			SlimPort:               SlimMessagingPort,
			SlimControllerEndpoint: slimControllerEndpoint,
			ServiceName:            serverName,
		}

		// Generate server config file
		outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.yaml", serverName))
		if err := GenerateConfigFromTemplate(ServerConfigTemplatePath, outputPath, data); err != nil {
			return fmt.Errorf("failed to generate config for server %s: %w", serverName, err)
		}

		// Generate connection config file
		outputPath = filepath.Join(outputDir, fmt.Sprintf("%s-conn-config.json", serverName))
		if err := GenerateConfigFromTemplate(ServerConnConfigTemplatePath, outputPath, data); err != nil {
			return fmt.Errorf("failed to generate config for server %s: %w", serverName, err)
		}

		fmt.Printf("Generated config for server '%s' at: %s\n", serverName, outputPath)
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
	fmt.Printf("Found %d servers in topology\n", len(topology.Topology.Servers))

	outputPath := "config/.gen"
	// Create the output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Generate server configs from topology
	err = GenerateServerConfigs(topology, slimControllerEndpoint, outputPath)
	if err != nil {
		log.Fatalf("Failed to generate server configs: %v", err)
	}

	fmt.Println("All configurations generated successfully!")
}
