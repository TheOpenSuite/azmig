package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// MigrateBoards handles the migration of boards and work items using the nkdAgility Azure DevOps Migration Tools.
func MigrateBoards(srcOrg, srcProj, srcTokn, trgtOrg, trgtProj, trgtTokn, typeMapping string, fullHistory bool) error {
	fmt.Printf("[INFO] Starting Boards/Work Items migration for project: %s\n", srcProj)

	// Parse type mapping string (e.g., "Task:Task,Bug:Issue")
	mappings := make(map[string]string)
	if typeMapping != "" {
		pairs := strings.Split(typeMapping, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) == 2 {
				mappings[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}

	// Configuration for the migration tool
	logLevel := "Information"
	if fullHistory {
		logLevel = "Debug"
	}

	// Replicating exactly what the provided Python-generated config does
	config := map[string]interface{}{
		"Serilog": map[string]interface{}{
			"MinimumLevel": logLevel,
		},
		"MigrationTools": map[string]interface{}{
			"Version": "16.0",
			"Endpoints": map[string]interface{}{
				"Source": map[string]interface{}{
					"$type":                    "TfsTeamProjectEndpoint",
					"EndpointType":             "TfsTeamProjectEndpoint",
					"Collection":               fmt.Sprintf("https://dev.azure.com/%s", srcOrg),
					"Project":                  srcProj,
					"AllowCrossProjectLinking": false,
					"ReflectedWorkItemIdField": "System.Id",
					"Authentication": map[string]interface{}{
						"AuthenticationMode": "AccessToken",
						"AccessToken":        srcTokn,
					},
				},
				"Target": map[string]interface{}{
					"$type":                    "TfsTeamProjectEndpoint",
					"EndpointType":             "TfsTeamProjectEndpoint",
					"Collection":               fmt.Sprintf("https://dev.azure.com/%s", trgtOrg),
					"Project":                  trgtProj,
					"AllowCrossProjectLinking": false,
					"ReflectedWorkItemIdField": "Microsoft.VSTS.Build.IntegrationBuild",
					"Authentication": map[string]interface{}{
						"AuthenticationMode": "AccessToken",
						"AccessToken":        trgtTokn,
					},
				},
			},
			"CommonTools": map[string]interface{}{
				"TfsUserMappingTool": map[string]interface{}{
					"Enabled": false,
				},
				"WorkItemTypeMappingTool": map[string]interface{}{
					"Enabled":  len(mappings) > 0,
					"Mappings": mappings,
				},
				"TfsAttachmentTool": map[string]interface{}{
					"Enabled":        true,
					"ExportBasePath": "C:\\temp\\WorkItemAttachmentExport",
					"MaxRevisions":   999999,
				},
				"TfsWorkItemTypeValidatorTool": map[string]interface{}{
					"Enabled": false,
				},
				"TfsRevisionManagerTool": map[string]interface{}{
					"Enabled":         fullHistory,
					"ReplayRevisions": fullHistory,
				},
				"StringManipulatorTool": map[string]interface{}{
					"Enabled": fullHistory,
				},
				"TfsNodeStructureTool": map[string]interface{}{
					"Enabled":                          true,
					"ReplicateAllExistingNodes":       true,
					"ShouldCreateMissingRevisionPaths": true,
					"Areas": map[string]interface{}{
						"Filters": []string{},
						"Mappings": []map[string]interface{}{
							{
								"Match":       fmt.Sprintf("^(?!%s)(.*)$", srcProj),
								"Replacement": trgtProj,
							},
						},
					},
					"Iterations": map[string]interface{}{
						"Filters": []string{},
						"Mappings": []map[string]interface{}{
							{
								"Match":       fmt.Sprintf("^(?!%s)(.*)$", srcProj),
								"Replacement": trgtProj,
							},
						},
					},
				},
			},
			"Processors": []interface{}{
				map[string]interface{}{
					"$type":         "TfsWorkItemMigrationProcessor",
					"ProcessorType": "TfsWorkItemMigrationProcessor",
					"Enabled":       true,
					"SourceName":    "Source",
					"TargetName":    "Target",
					"UpdateCreatedDate": fullHistory,
					"UpdateCreatedBy":   fullHistory,
					"FilterWorkItemsThatAlreadyExistInTarget": true,
					"WIQLQuery": fmt.Sprintf("SELECT [System.Id] FROM WorkItems WHERE [System.TeamProject] = @TeamProject AND [System.WorkItemType] NOT IN ('Test Suite', 'Test Plan','Shared Steps','Shared Parameter','Feedback Request') AND [System.AreaPath] UNDER '%s' ORDER BY [System.ChangedDate] desc", srcProj),
					"WorkItemCreateRetryLimit": 5,
					"GenerateMigrationComment": true,
					"ReplayRevisions":          false,
				},
			},
		},
	}

	// Write config to a temporary file
	configFile := "migrate_temp.json"
	configBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal migration config: %w", err)
	}

	if err := os.WriteFile(configFile, configBytes, 0644); err != nil {
		return fmt.Errorf("failed to write migration config file: %w", err)
	}
	defer os.Remove(configFile)

	// Execute the migration tool
	fmt.Println("[INFO] Executing devopsmigration...")
	cmd := exec.Command("devopsmigration", "execute", "--config", configFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("devopsmigration execution failed: %w", err)
	}

	fmt.Println("[SUCCESS] Boards/Work Items migration completed.")
	return nil
}
