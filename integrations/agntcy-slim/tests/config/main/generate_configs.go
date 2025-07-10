package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/agntcy/csit/integrations/agntcy-slim/tests/config"
)

const SlimEndpoint = "0.0.0.0:46357"
const SlimControllerEndpoint = "0.0.0.0:46358"

// SpireConfig represents the spire configuration section
type SpireConfig struct {
	Enabled bool `yaml:"enabled"`
}

// ServerConfigData represents the data structure for the server-config.tpl template
type ServerConfigData struct {
	Spire                  SpireConfig `yaml:"spire"`
	SlimEndpoint           string      `yaml:"slimEndpoint"`
	SlimControllerEndpoint string      `yaml:"slimControllerEndpoint"`
}

// GenerateServerConfig generates a server configuration file from the template
func GenerateServerConfig(templatePath, outputPath string, data ServerConfigData) error {
	// Parse the template file
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template file %s: %w", templatePath, err)
	}

	// Create the output directory if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
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
func GenerateServerConfigs(topology *config.Config, templatePath, outputDir string) error {
	// Generate a config file for each server
	for serverName, serverConfig := range topology.Topology.Servers {
		// Determine spire settings based on auth configuration
		spireEnabled := serverConfig.Auth.SpireMtls || serverConfig.Auth.SpireJwt

		// Create template data
		data := ServerConfigData{
			Spire: SpireConfig{
				Enabled: spireEnabled,
			},
			SlimEndpoint:           SlimEndpoint,
			SlimControllerEndpoint: SlimControllerEndpoint,
		}

		// Generate output file path
		outputPath := filepath.Join(outputDir, fmt.Sprintf("%s-config.yaml", serverName))

		// Generate the config file
		if err := GenerateServerConfig(templatePath, outputPath, data); err != nil {
			return fmt.Errorf("failed to generate config for server %s: %w", serverName, err)
		}

		fmt.Printf("Generated config for server '%s' at: %s\n", serverName, outputPath)
	}

	return nil
}

func main() {
	// Parse the fire-and-forget.yaml configuration
	topology, err := config.ParseTopology("agntcy-slim/config/fire-and-forget.yaml")
	if err != nil {
		log.Fatalf("Failed to parse configuration: %v", err)
	}

	fmt.Println("Configuration loaded successfully!")
	fmt.Printf("Found %d servers in topology\n", len(topology.Topology.Servers))

	// Generate server configs from topology
	templatePath := "agntcy-slim/config/server-config.tpl"
	outputDir := "agntcy-slim/config/.generated"

	// create outputDir if it does not exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	err = GenerateServerConfigs(topology, templatePath, outputDir)
	if err != nil {
		log.Fatalf("Failed to generate server configs: %v", err)
	}

	fmt.Println("All configurations generated successfully!")
}
