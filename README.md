# azmig

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Latest Release](https://img.shields.io/github/v/release/theopensuite/azmig?style=flat-square&color=orange)](https://github.com/your-username/azmig/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/theopensuite/azmig?style=flat-square)](https://github.com/your-username/azmig/blob/main/go.mod)
[![Build Status](https://github.com/theopensuite/azmig/actions/workflows/release.yml/badge.svg)](https://github.com/theopensuite/azmig/actions)

*azmig* is a Go-based CLI tool made primarly to centralize and partially automate migrations between different platforms like Azure DevOps, GitHub, and GitLab or same platform but different organizations. It can be used in mass repository migration, wiki transfers, and Azure-to-Azure board/work-item migrations.

## Key Features

* **Multi-Platform Support**: Migrate repositories from GitHub or GitLab to Azure DevOps (and vice-versa).
* **Wiki & Boards**: Support for migrating repository wikis and Azure DevOps work items/boards.
* **Repo Renaming**: Inline syntax for renaming repositories during the migration process.
* **Config Management**: Save your migration flags to JSON files and run batch migrations.

---
## Prerequisites
*azmig* acts as an orchestrator for existing CLI tools. Ensure the following are installed and in your `PATH`:
* **Git**: Required for all repository mirroring.
* **Azure CLI (`az`)**: Required for Azure DevOps operations.
* **GitHub CLI (`gh`)**: Required for GitHub source/target operations.
* **GitLab CLI (`glab`)**: Required for GitLab source/target operations.
* **DevOps Migration Tool**: Required for work item/board migrations.
You only need to download what you need and not every tool.
*Disclaimer: Devops Migration Tool can currently only be used with windows, so if you want to migrate any boards/work items, windows must be used.*

> **Tip**: Run `azmig verify` to check your local environment for these dependencies.

---
## Installation

**Option 1**
```bash
go build -o azmig .
mv azmig /usr/local/bin/ # Or add to your PATH
```

Option 2:
```bash
go run ./main.go ./boards_migrator.go [command]
```

---
## Usage Guide

### 1. Verify Environment
Check if you have the necessary tools installed:
```bash
azmig verify
```

### 2. List Repositories
Explore the source platform before migrating:
```bash
azmig list --plat github --org my-org --tokn $GH_TOKEN
```
> A slight note: It is not recommended to use a token in the terminal if the device can be used or shared by others, use an environmental variable instead.
### 3. Run a Migration
Migrate a specific repository from GitHub to Azure DevOps:

```bash
azmig run \
  --src-plat github --src-org MyGithubOrg --src-proj "NotUsedInGH" \
  --trgt-plat azure --trgt-org MyAzureOrg --trgt-proj MyTargetProject \
  --repo "my-repo"
```
Not every flag is required, as some default to certain values and others consume from another flag.
For example: The src and trgt platform are defaulted to azure, and if trgt-tokn is not given, it assumes same organization and uses source token instead,
#### Advanced Migration Options:
* **Rename a Repo**: `--repo "old-name:new-name"`
* **Migrate All**: `--repo "MIGRATEALL"`
* **Include Wikis**: Add the `--wiki` or `-w` flag.
* **Include Boards**: Add `--boards` (Azure to Azure only).

---
## Configuration & Automation

### Saving Configurations
If you have a complex migration, you can save your flags to a JSON file:
```bash
azmig run [FLAGS] --config
```
This creates a file in `./config/your-target-project.json`. The config is saved by the target project's name.

### Loading Configurations
Run one or multiple saved migrations:
```bash
azmig load my-project-config
azmig load config1 config2 config3
```

---
## Environment Variables
To avoid passing tokens in plain text, *azmig* supports the following environment variables:

| Variable               | Description              |
| :--------------------- | :----------------------- |
| `AZMIG_SRC_TOKEN`      | Global source token      |
| `AZURE_DEVOPS_EXT_PAT` | Default for Azure DevOps |
| `GH_TOKEN`             | Default for GitHub       |
| `GITLAB_TOKEN`         | Default for GitLab       |

---
## Command Reference

| Command     | Description                                                          |
| :---------- | :------------------------------------------------------------------- |
| `verify`    | Validates that `git`, `az`, `gh`, etc., are installed.               |
| `run`       | The primary migration engine.                                        |
| `list`      | Lists repositories for a given organization/platform.                |
| `load`      | Executes migration(s) based on saved JSON config files.              |
| `--verbose` | Enable debug logging to see internal commands and temp folder paths. |
| `--help`    | Use for more information and how to use it.                          |

---
config/sample-migration.json
```bash
{
  "SrcPlat": "github",
  "SrcOrg": "my-source-github-org",
  "SrcProj": "",
  "SrcTokn": "",
  "Repo": "original-repo:new-name,another-repo,third-repo:renamed-target",
  "TrgtPlat": "azure",
  "TrgtOrg": "ABCSoftware",
  "TrgtProj": "ProjectD",
  "TrgtTokn": "",
  "Wiki": true,
  "Boards": false,
  "TypeMapping": "Task:Task,Bug:Issue",
  "Config": false
}
```

---
## Contributing

1. Fork the repository.
2. Create your feature branch.
3. Commit your changes.
4. Push to the branch.
5. Open a Pull Request.
