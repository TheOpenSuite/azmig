package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"text/tabwriter"

	"github.com/alecthomas/kong"
)

var Version = "0.0.0-src"

type CLI struct {
	Verbose bool `help:"Show extra info." short:"V"`
	Version kong.VersionFlag `help:"Shows the current version." short:"v"`

	// Our Ghost Router
	Default DefaultHandler `cmd:"" default:"1" hidden:""`
	Verify  VerifyC        `cmd:"" help:"Verifies all needed packages."`
	Run     RunC           `cmd:"" help:"Used to run the migration process."`
	List    ListC          `cmd:"" help:"Lists all the Repositories."`
	Load    LoadC          `cmd:"" help:"Loads config file(s) and runs the migration."`
}

// Ghost router
type DefaultHandler struct{}

func (d *DefaultHandler) BeforeApply(ctx *kong.Context) error {
	// 1. Block explicit "default" call
	for _, arg := range os.Args {
		if arg == "default" {
			return fmt.Errorf("command 'default' is restricted and can't be used.")
		}
	}

	if len(os.Args) < 2 {
		return d.triggerStandardError(ctx)
	}

	return nil
}

func (d *DefaultHandler) triggerStandardError(ctx *kong.Context) error {
	// Dynamically find all commands that aren't hidden
	var visible []string
	for _, cmd := range ctx.Model.Node.Children {
		if !cmd.Hidden {
			visible = append(visible, cmd.Name)
		}
	}
	return fmt.Errorf("expected one of: %s", strings.Join(visible, ", "))
}

func (d *DefaultHandler) Run(cli *CLI) error {
	// Flags to run in root without a command
	// The section below is commented out as it was only done for a test.
	/*if cli.Version {
		fmt.Println("azmig version 1.0.0")
	}*/
	return nil
}

// Commands 

// Normal flags router for all commands
func (c *CLI) BeforeRun() error {
	if c.Verbose {
		fmt.Println("[DEBUG] Verbose mode active")
	}
	return nil
}

// Normal commands
type VerifyC struct{}

func (v *VerifyC) Run() error {
	Tools := []string{"git", "az", "gh", "glab", "devopsmigration"}
	var flag string
	for _, Tool := range Tools {
		if Tool == "devopsmigration" {
			flag = "--help"
		} else {
			flag = "--version"
		}
		cmd := exec.Command(Tool, flag)
		_, err := cmd.CombinedOutput()
		if err != nil {
			switch Tool {
			case "gh":
				fmt.Println("[Optional] GitHub CLI (gh) is missing. For more information (https://cli.github.com/)")

			case "az":
				fmt.Println("Azure CLI (az) is missing. For more information (https://learn.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest)")

			case "git":
				fmt.Println("Git is missing. For more information (https://git-scm.com/install/)")

			case "glab":
				fmt.Println("[Optional] GitLab (glab) is missing. For more information (https://gitlab.com/gitlab-org/cli#installation)")

			default:
				fmt.Println("[Optional] ", Tool, "is either missing or not found in PATH. This tool is used to migrate work items and boards but only supports windows machines, for more infomation (https://devopsmigration.io/docs/setup/installation/)")
			}
			continue
		}
		fmt.Println(Tool, "is installed.")
	}
	return nil
}

type RunC struct {
	SrcPlat     string `help:"Source platform (e.g., 'github')." name:"src-plat" default:"azure" enum:"azure, github, gitlab"`
	SrcOrg      string `help:"Source organization (e.g., 'myOrganization')." name:"src-org" required:""`
	SrcProj     string `help:"Source project (e.g. 'myProject').," name:"src-proj" required:""`
	SrcTokn     string `help:"Source Personal Access Token (PAT)." name:"src-tokn" env:"AZMIG_SRC_TOKEN"`
	Repo     		string `help:"Source project repo(s). To migrate all, type 'MIGRATEALL'. To rename a repo -> (e.g., 'original:custom')" name:"repo" short:"r"`
	TrgtPlat    string `help:"Target platform" name:"trgt-plat" default:"azure" enum:"azure, github, gitlab"`
	TrgtOrg     string `help:"Target organization. Defaults to source organization if empty" name:"trgt-org"`
	TrgtProj    string `help:"Target project" name:"trgt-proj" required:""`
	TrgtTokn    string `help:"Target Personal Access Token (PAT). Defaults to source token if empty." name:"trgt-tokn"`
	Wiki        bool   `help:"Migrates the wiki." optional:"" short:"w"`
	Boards      bool   `help:"Migrates the boards and work items. AzuretoAzure only." optional:"" short:"b"`
	TypeMapping string `help:"Work item type mapping (e.g., 'Task:Task,Bug:Issue')." name:"type-mapping" short:"m"`
	Config			bool   `help:"Saves flags to a JSON file named after the target project. Can be used to save your config or run multiple configs at once." short:"c" optional:""`
}

func (r *RunC) Run(cli *CLI) error {
	
	if r.Config {
      safeName := strings.ReplaceAll(strings.ToLower(r.TrgtProj), " ", "-")
      if safeName == "" {
          return fmt.Errorf("target project name is required to create a config file")
      }
      os.MkdirAll("config", 0755)
      fileName := fmt.Sprintf("config/%s.json", safeName)

      if _, err := os.Stat(fileName); err == nil {
          fmt.Printf("Config file '%s' already exists. Overwrite? (y/n): ", fileName)
          var confirm string
          fmt.Scanln(&confirm)
          if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
              fmt.Println("Save aborted.")
              return nil
          }
      }

      jsonData, _ := json.MarshalIndent(r, "", "  ")
      if err := os.WriteFile(fileName, jsonData, 0644); err != nil {
          return fmt.Errorf("failed to save config: %w", err)
      }
			fmt.Printf("Config saved to %s. To start it, either remove the config flag or with the command: 'azmig load %s'\n", fileName, safeName)
      return nil
  }

	if r.SrcTokn == "" {
		if r.SrcPlat == "azure" {
			r.SrcTokn = os.Getenv("AZURE_DEVOPS_EXT_PAT")
		} else if r.SrcPlat == "github" {
			r.SrcTokn = os.Getenv("GH_TOKEN")
		} else if r.SrcPlat == "gitlab" {
			r.SrcTokn = os.Getenv("GITLAB_TOKEN")
		}
	}

	if r.SrcTokn == "" {
		return fmt.Errorf("Token couldn't be found in either the flag or an environmental variable. Ensure to either use the '--src-tokn' or a shell environemtnal variable.")
	}

	if r.TrgtTokn == "" {
		fmt.Printf("[INFO] Target token not given, assuming source token as target token.\n")
		r.TrgtTokn = r.SrcTokn
	}
	
	if r.TrgtOrg == "" {
		fmt.Printf("[INFO] Target organization not given, assuming source organization.\n")
		r.TrgtOrg = r.SrcOrg
	}

	if (r.Boards || r.TypeMapping != "") && (r.SrcPlat != "azure" || r.TrgtPlat != "azure") {
		return fmt.Errorf("boards migration (--boards) and type mapping (--type-mapping) are currently only supported for azure-to-azure migrations")
	}

	if err := r.migrateRepo(cli); err != nil {
		return err
	}


	return nil
}

func (r *RunC) migrateRepo(cli *CLI) error {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	folderID := r1.Intn(90000) // Random number generator for the tmp folder
	baseTempDir := os.TempDir()

	// Different sources and their commands logic
	var cmd *exec.Cmd
	switch strings.ToLower(r.SrcPlat) {
	case "azure":
		vsrcorg := fmt.Sprintf("https://dev.azure.com/%s", r.SrcOrg)
		os.Setenv("AZURE_DEVOPS_EXT_PAT", r.SrcTokn)
		if cli.Verbose {
            fmt.Printf("[DEBUG] Listing Azure repos for project: %s in org: %s\n", r.SrcProj, vsrcorg)
      }
		cmd = exec.Command("az", "repos", "list", "--project", r.SrcProj, "--org", vsrcorg, "--output", "json", "--query", "[].name")
	case "github":
		os.Setenv("GH_TOKEN", r.SrcTokn)
		if cli.Verbose {
            fmt.Printf("[DEBUG] Listing GitHub repos for org: %s\n", r.SrcOrg)
      }
		cmd = exec.Command("gh", "repo", "list", r.SrcOrg, "--limit", "1000", "--json", "name", "--jq", ".[].name")
	case "gitlab":
		os.Setenv("GITLAB_TOKEN", r.SrcTokn)
		if cli.Verbose {
            fmt.Printf("[DEBUG] Listing GitHub repos for org: %s\n", r.SrcOrg)
      }
		cmd = exec.Command("glab", "repo", "list", "-P", r.SrcOrg, "--output", "json", "--query", "[].name")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list source repos: %w\nOutput: %s", err, string(output))
	}

	var allSourceRepos []string
	if err := json.Unmarshal(output, &allSourceRepos); err != nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, l := range lines {
			if strings.TrimSpace(l) != "" {
				allSourceRepos = append(allSourceRepos, strings.TrimSpace(l))
			}
		}
	}


	var reposToMigrate []string
    if r.Repo == "MIGRATEALL" {
        reposToMigrate = allSourceRepos
    } else {
        parts := strings.Split(r.Repo, ",")
        for _, p := range parts {
            clean := strings.Trim(strings.TrimSpace(p), "'\"")
            if clean != "" {
                reposToMigrate = append(reposToMigrate, clean)
            }
        }
    }

    fmt.Println("\nMigration Plan:")
    for i, entry := range reposToMigrate {
        src, trgt := entry, entry
        if strings.Contains(entry, ":") {
            kv := strings.SplitN(entry, ":", 2)
            src = strings.TrimSpace(kv[0])
            trgt = strings.TrimSpace(kv[1])
        }

        status := ""
        if src != trgt {
            status = " (RENAME)"
        }
        fmt.Printf("[%d] %s -> %s%s\n", i, src, trgt, status)
    }


	fmt.Print("\nProceed with migration? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
	    return fmt.Errorf("migration aborted by user")
	}

	// Migration loop
	for _, entry := range reposToMigrate {
      srcBase, trgtBase := entry, entry
      if strings.Contains(entry, ":") {
          kv := strings.SplitN(entry, ":", 2)
          srcBase = strings.TrimSpace(kv[0])
          trgtBase = strings.TrimSpace(kv[1])
      }

      suffixes := []string{""}
      if r.Wiki {
          suffixes = append(suffixes, "WIKI_MARKER")
      }

      for _, suffixMarker := range suffixes {
          var srcName, trgtName string

          if suffixMarker == "WIKI_MARKER" {
              if strings.ToLower(r.SrcPlat) == "azure" {
                  srcName = srcBase + ".wiki"
              } else {
                  srcName = srcBase + ".wiki.git"
              }

              if strings.ToLower(r.TrgtPlat) == "azure" {
                  trgtName = trgtBase + ".wiki"
              } else {
                  trgtName = trgtBase + ".wiki.git"
              }
          } else {
              srcName = srcBase
              trgtName = trgtBase
          }	
	

			var checkCmd, createCmd *exec.Cmd
			var srcurl, trgturl string

			// Target setup switch
			switch strings.ToLower(r.TrgtPlat) {
			case "azure":
				vtrgtorg := fmt.Sprintf("https://dev.azure.com/%s", url.PathEscape(r.TrgtOrg))
				os.Setenv("AZURE_DEVOPS_EXT_PAT", r.TrgtTokn)
				checkCmd = exec.Command("az", "repos", "show", "--repository", trgtName, "--project", r.TrgtProj, "--org", vtrgtorg)
				createCmd = exec.Command("az", "repos", "create", "--name", trgtName, "--project", r.TrgtProj, "--org", vtrgtorg)
				trgturl = fmt.Sprintf("https://:%s@dev.azure.com/%s/%s/_git/%s", r.TrgtTokn, url.PathEscape(r.TrgtOrg), url.PathEscape(r.TrgtProj), url.PathEscape(trgtName))
			case "github":
				os.Setenv("GH_TOKEN", r.TrgtTokn)
				checkCmd = exec.Command("gh", "repo", "view", fmt.Sprintf("%s/%s", r.TrgtOrg, trgtName))
				createCmd = exec.Command("gh", "repo", "create", fmt.Sprintf("%s/%s", r.TrgtOrg, trgtName), "--private")
				trgturl = fmt.Sprintf("https://%s@github.com/%s/%s.git", r.TrgtTokn, r.TrgtOrg, trgtName)
			case "gitlab":
				os.Setenv("GITLAB_TOKEN", r.TrgtTokn)
				checkCmd = exec.Command("glab", "repo", "view", fmt.Sprintf("%s/%s", r.TrgtOrg, trgtName))
				createCmd = exec.Command("glab", "repo", "create", trgtName, "--group", r.TrgtOrg, "--private")
				trgturl = fmt.Sprintf("https://oauth2:%s@gitlab.com/%s/%s.git", r.TrgtTokn, r.TrgtOrg, trgtName)
			}

			// Source mapping
			switch strings.ToLower(r.SrcPlat) {
			case "azure":
				srcurl = fmt.Sprintf("https://:%s@dev.azure.com/%s/%s/_git/%s", r.SrcTokn, url.PathEscape(r.SrcOrg), url.PathEscape(r.SrcProj), url.PathEscape(srcName))
			case "github":
				srcurl = fmt.Sprintf("https://%s@github.com/%s/%s.git", r.SrcTokn, r.SrcOrg, srcName)
			case "gitlab":
				srcurl = fmt.Sprintf("https://oauth2:%s@gitlab.com/%s/%s.git", r.SrcTokn, r.SrcOrg, srcName)
			}

			folderName := fmt.Sprintf("tmp_folder_%d", folderID)
			tmpFolder := filepath.Join(baseTempDir, folderName)
			folderID++

			if cli.Verbose {
          fmt.Printf("[DEBUG] Source URL: %s\n", strings.ReplaceAll(srcurl, r.SrcTokn, "********"))
          fmt.Printf("[DEBUG] Target URL: %s\n", strings.ReplaceAll(trgturl, r.TrgtTokn, "********"))
          fmt.Printf("[DEBUG] Using temp folder: %s\n", tmpFolder)
        }

			// Check if it exists, if not, then create
			if err := checkCmd.Run(); err == nil {
				fmt.Printf("[WARNING] '%s' exists in target. Override? (y/n): ", trgtName)
				var resp string
				fmt.Scanln(&resp)
				if strings.ToLower(strings.TrimSpace(resp)) != "y" {
					fmt.Printf("%s did not migrate.\n", trgtName)
					continue
				}
			} else {
				if err := createCmd.Run(); err != nil {
					if suffixMarker == "WIKI_MARKER" {
						fmt.Printf("Skipping Wiki for %s (Target creation failed/not supported)\n", srcBase)
						continue // Wikis fail creation if source doesn't have one
					}
					return fmt.Errorf("failed to create repo %s: %w", trgtName, err)
				}
			}

			// Perform mirror
			fmt.Printf("Mirroring %s -> %s...\n", srcName, trgtName)
			output, err = exec.Command("git", "clone", "--mirror", srcurl, tmpFolder).CombinedOutput()

			if cli.Verbose && len(output) > 0 {
				fmt.Printf("[DEBUG] %s\n", string(output))
			}

			if err != nil {
				os.RemoveAll(tmpFolder)
				if suffixMarker == "WIKI_MARKER" {
					fmt.Printf("No Wiki found for %s on source platform. Skipping.\n", srcBase)
					continue
				}
				return fmt.Errorf("failed to clone %s: %w", srcName, err)
			}


			// Push branches
			pushBranches := exec.Command("git", "push", trgturl, "--all")
			pushBranches.Dir = tmpFolder
			output, err = pushBranches.CombinedOutput()

			if cli.Verbose && len(output) > 0 {
    			fmt.Printf("[DEBUG] Git Push (Branches) Output:\n%s\n", string(output))
			}

			if err != nil {
    			os.RemoveAll(tmpFolder)
    			return fmt.Errorf("failed to push branches for %s: %w\nDetails: %s", trgtName, err, string(output))
			}

			// push tags
			pushTags := exec.Command("git", "push", trgturl, "--tags")
			pushTags.Dir = tmpFolder
			output, err = pushTags.CombinedOutput()

			if cli.Verbose && len(output) > 0 {
    			fmt.Printf("[DEBUG] Git Push (Tags) Output:\n%s\n", string(output))
			}

			if err != nil {
    			os.RemoveAll(tmpFolder)
    			return fmt.Errorf("failed to push tags for %s: %w\nDetails: %s", trgtName, err, string(output))
			}

			if cli.Verbose {
				fmt.Printf("[DEBUG] Deleting %s...\n", folderName)
			}
			os.RemoveAll(tmpFolder)
			fmt.Printf("[SUCCESS] Migrated %s\n", srcName)
		}
	}

	// Boards migration check and call
	if r.Boards && r.SrcPlat == "azure" && r.TrgtPlat == "azure" {
		if err := MigrateBoards(r.SrcOrg, r.SrcProj, r.SrcTokn, r.TrgtOrg, r.TrgtProj, r.TrgtTokn, r.TypeMapping); err != nil {
			return err
		}
	}
	return nil
}

type ListC struct {
	Plat  				 	 string `help:"Platform name." name:"plat" default:"azure" enum:"azure, github, gitlab"`
	Org   				 	 string `help:"Organization name." name:"org" required:""`
	Proj  				 	 string `help:"Project name." name:"proj" required:""`
	Tokn  				 	 string `help:"The Personal Access Token (PAT)." name:"tokn" env:"AZMIG_SRC_TOKEN" required:""`
	MappingReference bool		`help:"Shows the work items mapping reference as an extra." short:"e"`
}

func (r *ListC) Run(cli *CLI) error {
    switch strings.ToLower(r.Plat) {
    case "azure":
        vsrcorg := fmt.Sprintf("https://dev.azure.com/%s", r.Org)
        os.Setenv("AZURE_DEVOPS_EXT_PAT", r.Tokn)
        cmd := exec.Command("az", "repos", "list", "--project", r.Proj, "--org", vsrcorg, "--output", "json", "--query", "[].name")
        runAndPrint(cmd, "Azure DevOps")

    case "github":
        os.Setenv("GITHUB_TOKEN", r.Tokn)
        cmd := exec.Command("gh", "repo", "list", r.Org, "--json", "name", "--jq", "[].name")
        runAndPrint(cmd, "GitHub")

    case "gitlab":
        os.Setenv("GITLAB_TOKEN", r.Tokn)
        cmd := exec.Command("glab", "repo", "list", "-G", r.Org, "--output", "json")
        runAndPrint(cmd, "GitLab")

    default:
        return fmt.Errorf("unsupported platform: %s", r.Plat)
    }
		if r.MappingReference {
    	PrintProcessMappingTable()
		}

    return nil
}

func runAndPrint(cmd *exec.Cmd, platform string) {
    output, err := cmd.CombinedOutput()
    if err != nil {
        fmt.Printf("Error fetching %s repos: %v\nOutput: %s\n", platform, err, string(output))
        return
    }
    fmt.Printf("\n%s Repositories\n", platform)
    fmt.Println(string(output))
}

// renders the mapping table
func PrintProcessMappingTable() {
    w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

    fmt.Println("\nAZURE DEVOPS PROCESS REFERENCE")
    fmt.Println(strings.Repeat("-", 80))

    // Header
    fmt.Fprintln(w, "HIERARCHY\tBASIC\tAGILE\tSCRUM\tCMMI")
    fmt.Fprintln(w, "---------\t-----\t-----\t-----\t----")

    // Portfolio levels
    fmt.Fprintln(w, "Portfolio\tEpic\tEpic\tEpic\tEpic")
    fmt.Fprintln(w, "Portfolio\t-\tFeature\tFeature\tFeature")

    // Backlog level
    fmt.Fprintln(w, "Backlog\tIssue\tUser Story\tProduct Backlog Item\tRequirement")

    // Iteration level
    fmt.Fprintln(w, "Work\tTask\tTask\tTask\tTask")

    // Tracking & management
    fmt.Fprintln(w, "Tracking\t-\tIssue\tImpediment\tIssue")
    fmt.Fprintln(w, "Project Mgmt\t-\t-\t-\tReview")
    fmt.Fprintln(w, "Project Mgmt\t-\t-\t-\tRisk")
    fmt.Fprintln(w, "Changes\t-\t-\t-\tChange Request")

    // Defects & quality
    fmt.Fprintln(w, "Defect\t-\tBug\tBug\tBug")
    fmt.Fprintln(w, "Testing\tTest Case\tTest Case\tTest Case\tTest Case")
    fmt.Fprintln(w, "Testing\tTest Plan\tTest Plan\tTest Plan\tTest Plan")
    fmt.Fprintln(w, "Testing\tTest Suite\tTest Suite\tTest Suite\tTest Suite")

    // Shared ssets
    fmt.Fprintln(w, "Assets\tShared Step\tShared Step\tShared Step\tShared Step")
    fmt.Fprintln(w, "Assets\tShared Param\tShared Param\tShared Param\tShared Param")

    w.Flush()
    fmt.Println(strings.Repeat("-", 80))
		fmt.Println("Note: This mapping reference may change or not be 100% accurate.\n")
}

type LoadC struct {
	Files []string `arg:"" help:"Path to the config file(s)." required:""`
}

func (l *LoadC) Run(cli *CLI) error {

		for _, file := range l.Files {
        cleanName := strings.TrimSuffix(strings.ToLower(file), ".json")
        filePath := filepath.Join("config", cleanName+".json")

        if _, err := os.Stat(filePath); os.IsNotExist(err) {
            return fmt.Errorf("Config file '%s' not found in config/ folder", cleanName)
        }
    }
		
		for _, file := range l.Files {
			// Strip .json if typed it, then forces it back on to allow just the config name to be entered.
    	cleanName := strings.TrimSuffix(strings.ToLower(file), ".json")
			fileData, _ := os.ReadFile(filepath.Join("config", cleanName+".json"))

    	var r RunC
    	if err := json.Unmarshal(fileData, &r); err != nil {
        	return fmt.Errorf("Failed to parse config %s: %w", cleanName, err)
    	}

    	r.Config = false

			if err := r.Run(cli); err != nil {
				fmt.Printf("Migration failed for %s: %v\n", file, err)
				fmt.Print("Do you want to continue? (y/n): ")
				var resp string
          	fmt.Scanln(&resp)
          	if strings.ToLower(strings.TrimSpace(resp)) != "y" {
              	fmt.Println("Migration has been halted.")
              	return nil
          	}

			}

		}
    		
		fmt.Println("\nAll migrations have finished.")
    return nil

}

// Main

func main() {
	cli := &CLI{}
	ctx := kong.Parse(cli,
		kong.Name("azmig"),
		kong.Description("An Azure migration tool, currently supporting github and gitlab for general repo migration. \nThe idea is to centralize the general migrations between different tools and mass migrating multipe repos with a single command."),
		kong.UsageOnError(),
		kong.Vars{
			"version": Version,
		},
	)

	ctx.FatalIfErrorf(ctx.Run(cli))
}
