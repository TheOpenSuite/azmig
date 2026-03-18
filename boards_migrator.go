package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Config structure
type MigrationConfig struct {
	MigrationTools MigrationTools `json:"MigrationTools"`
}

type MigrationTools struct {
	Endpoints   Endpoints   `json:"Endpoints"`
	CommonTools CommonTools `json:"CommonTools"`
	Processors  []Processor `json:"Processors"`
}

type Endpoints struct {
	Source Endpoint `json:"Source"`
	Target Endpoint `json:"Target"`
}

type Endpoint struct {
	Collection     string         `json:"Collection"`
	Project        string         `json:"Project"`
	Authentication Authentication `json:"Authentication"`
}

type Authentication struct {
	AccessToken string `json:"AccessToken"`
}

type CommonTools struct {
	WorkItemTypeMappingTool WorkItemTypeMappingTool `json:"WorkItemTypeMappingTool"`
}

type WorkItemTypeMappingTool struct {
	Enabled  bool              `json:"Enabled"`
	Mappings map[string]string `json:"Mappings"`
}

type Processor struct {
	Type    string `json:"$type"`
	Enabled bool   `json:"Enabled"`
}

// Handles the migration of boards and work items using the devopsmigration tool.
func MigrateBoards(srcOrg, srcProj, srcToken, trgtOrg, trgtProj, trgtToken, typeMapping string) error {
	fmt.Println("[BOARDS] Starting boards and work items migration...")

	mappings := make(map[string]string)
	if typeMapping != "" {
		pairs := strings.Split(typeMapping, ",")
		for _, pair := range pairs {
			kv := strings.Split(pair, ":")
			if len(kv) == 2 {
				mappings[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}

	// Creates the config
	cfg := MigrationConfig{
		MigrationTools: MigrationTools{
			Endpoints: Endpoints{
				Source: Endpoint{
					Collection:     fmt.Sprintf("https://dev.azure.com/%s", srcOrg),
					Project:        srcProj,
					Authentication: Authentication{AccessToken: srcToken},
				},
				Target: Endpoint{
					Collection:     fmt.Sprintf("https://dev.azure.com/%s", trgtOrg),
					Project:        trgtProj,
					Authentication: Authentication{AccessToken: trgtToken},
				},
			},
			CommonTools: CommonTools{
				WorkItemTypeMappingTool: WorkItemTypeMappingTool{
					Enabled:  true,
					Mappings: mappings,
				},
			},
			Processors: []Processor{
				{
					Type:    "TfsWorkItemMigrationProcessor",
					Enabled: true,
				},
			},
		},
	}

	cfgJSON, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal migration config: %w", err)
	}

	configPath := "tm.json"
	if err := os.WriteFile(configPath, cfgJSON, 0644); err != nil {
		return fmt.Errorf("failed to write migration config file: %w", err)
	}
	defer os.Remove(configPath)

	fmt.Println("[BOARDS] Executing devopsmigration...")
	migrationCmd := exec.Command("devopsmigration", "execute", "--config", configPath)
	migrationCmd.Stdout = os.Stdout
	migrationCmd.Stderr = os.Stderr
	if err := migrationCmd.Run(); err != nil {
		return fmt.Errorf("devopsmigration failed: %w", err)
	}
	fmt.Println("[SUCCESS] Boards migration completed.")
	return nil
}
