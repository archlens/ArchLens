# ArchLensGo <!-- omit in toc -->

<p align="center">
<a href="https://github.com/archlens/ArchLensGo/actions/workflows/test.yml"> <img alt="GitHub Actions Workflow Status Tests" src="https://img.shields.io/github/actions/workflow/status/archlens/ArchLensGo/test.yml?label=tests"></a>
<a href="https://github.com/archlens/ArchLensGo/actions/workflows/build_release.yml"> <img alt="GitHub Actions Workflow Status Build" src="https://img.shields.io/github/actions/workflow/status/archlens/ArchLensGo/build_release.yml"></a>
</p>

Go rewrite of the classic command line tool, that generates architectural views from lightweight specifications

ArchLensGo generates customizable visual internal dependency diagrams of your codebase, showing packages and their dependencies. It currently supports Python and Go, but also supports creation of custom language parsers (easily done with AI, please see spec in docs folder). ArchLens also has a dedicated Visual Studio Code extension, you can call it through CLI and can highlight the differences between GitHub branches to make pull request reviews easier (CURRENTLY NOT SUPPORTED BY ArchLensGo as the input format needs to be updated).

## Table of Contents  <!-- omit in toc -->

- [Installation](#installation)
  - [Homebrew](#homebrew)
  - [Binary Install](#binary-install)
  - [Go Install](#go-install)
- [Configuration](#configuration)
  - [Configuration Reference](#configuration-reference)
- [Commands](#commands)
- [Defining Views](#defining-views)
- [Diff Views](#diff-views)

## Installation

### Homebrew

Arhclens has it's own homebrew tap which you can use to install the Archlens tool by using the below command.

```console
brew tap archlens/tap
brew install archlens/tap/archlens
```

### Install Script

The Archlens tool comes with an install script hosted on this repository. You can run it using this command. *!Please always look at scripts before running them on your machine!*

```console
curl -L -o install.sh https://github.com/archlens/ArchLensGo/raw/refs/heads/main/installer/install.sh && chmod +x install.sh && ./install.sh
```

### Binary Install

The release page contains pre-packaged binaries built for MacOS, Linux and Windows (both for arm and x86 based systems). You can install these in your preferred bin location.

### Go Install

*COMING SOON*

## Configuration

ArchLens is configured through an `archlens.json` file in the root of your project. To generate a template use the `init` command outlined in the commands chapter. 

### Configuration Reference

| Field | Required | Description |
|---|---|---|
| `name` | Yes | Name of your project |
| `rootFolder` | Yes | Path to your source root relative to the project root (e.g. `"src"`) |
| `github.url` | WIP | URL of the GitHub repository |
| `github.branch` | WIP | The base branch to compare against (e.g. `"main"`) |
| `saveLocation` | No | Where to save generated diagrams. Defaults to `"./diagrams/"` |
| `runCommand` | Yes | What command to run the AST language parsers with. Default is provided by init command |
| `views.<viewname>.include[]` | Yes | Which files to include in the named view `<viewname>` defined as a list of glob pattern strings |
| `views.<viewname>.exclude[]` | Yes | Which files to exclude in the named view `<viewname>` defined as a list of glob pattern strings |

Below is an example configuration for a Go project. It is however recommended to use absolute paths for the rootFolder and runCommand as to cause as few issues as possible.

```json
{
    "name": "Archlens",
    "rootFolder": "../src",
    "github": {
        "url": "https://github.com/archlens/ArchLens",
        "branch": "master"
    },
    "saveLocation": "./diagrams/",
    "runCommand": "go run ./ast_parsers/go/main.go",
    "views": {
        "completeView": {
            "include": [
                "**/*.*",
                "*.*"
            ],
            "exclude": [
                "go.sum",
                "go.mod",
                "**/.archlens/**",
                "**/.archlens"
            ]
        },
        "smallerView": {
            "include": [
                "cmd/*.*",
                "main.go"
            ],
            "exclude": [
                "**/.archlens/**",
                "**/.archlens"
            ]
        }
    }
}
```

## Commands

| Command | Description |
|---|---|
| `archlens init` | Creates the `archlens.json` config template |
| `archlens init <language>` | Creates the aforementioned `archlens.json`, as well as creating the language specific ast_parser file |
| `archlens render [/path/to/archlens.json]` | Renders all views defined in the config using the supplied `archlens.json` defaults to `.` |
| `archlens render-diff [/path/to/archlens.json]` | Renders difference views comparing current branch to the base branch NOT SUPPORTED YET |
| `archlens render-diff <base-branch> <compare-branch> | Renders difference views between base branch and compare branch |

## Defining Views

Views control what is shown in each diagram. Each view is a named entry under `"views"` in your config as described in the table above.

*EXAMPLE VIEWS COMING SOON*

## Diff Views

Diff views highlight dependency changes between your current branch and the base branch specified in `github.branch`. Changed elements are shown in **green** (added) and **red** (removed).

Make sure you are on a feature branch (not the base branch), then run:

```console
archlens render-diff
```

If there are no differences, a diagram without highlights is still generated.

Diff output indicates:

- **Green package/arrow** — added in the current branch
- **Red package/arrow** — removed in the current branch
- Count additive changes on arrows **Green arrow** (e.g. `5 (+2)`)
- Count subtractive changes on arrows **Red arrow** (e.g. `5 (-2)`)

