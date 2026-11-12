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

### Go Install

ArchLens is hosted as a package on [https://pkg.go.dev/](https://pkg.go.dev/) meaning you can use go install to install the tool as a binary. Please keep in mind that the go bin folder is usually not auto included in `$PATH` so you might need to add that folder to your path env variable.

```console
go install github.com/archlens/archlens
```

### Binary Install

The release page contains pre-packaged binaries built for MacOS, Linux and Windows (both for arm and x86 based systems). You can install these in your preferred bin location.

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

Below is an example configuration for a Go project.

```json
{
    "name": "Archlens",
    "rootFolder": "./src",
    "github": {
        "url": "https://github.com/archlens/ArchLens",
        "branch": "main"
    },
    "saveLocation": "./diagrams/",
    "runCommand": "go run ../ast_parsers/go/ast_parser.go",
    "views": {
        "completeView": {
            "include": [
                "**/*.go",
                "*.go"
            ],
            "exclude": [
                "go.sum",
                "go.mod"
            ]
        },
        "smallerView": {
            "include": [
                "cmd/*.*",
                "main.go"
            ],
            "exclude": []
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
| `archlens render-diff <base-branch> <compare-branch>` | Renders difference views between base branch and compare branch |

## Defining Views

Views control what is shown in each diagram. Each view is a named entry under `"views"` in your config as described in the table above. You can see 2 different views described in the example config above. The completeView is supposed to grab everything where as the smaller view only looks at the cmd folder and main.go. Everything is defined as glob patterns so if you want certain folders or files included you can write a glob pattern that matches those constraints.

## Diff Views

Diff views highlight dependency changes between your current branch and the base branch specified in `github.branch`. Changed elements are shown in **green** (added) and **red** (removed).

Make sure you are on a feature branch (not the base branch), then run:

```console
archlens render-diff <compare-branch>
```

If there are no differences, a diagram without highlights is still generated.

Diff output indicates:

- **Green package/arrow** — added in the current branch
- **Red package/arrow** — removed in the current branch
- Count additive changes on arrows **Green arrow** (e.g. `5 (+2)`)
- Count subtractive changes on arrows **Red arrow** (e.g. `5 (-2)`)

## Custom Parsers

The tool accepts custom parsers as long as it adheres the IO specifications described in [AST_spec.md](docs/AST_spec.md). Users can try giving the spec to an LLM and ask it to generate an AST parser in the language that is trying to be analyzed using the specification given. The user can then use the init command to generate an empty archlens.json file and fill out the runCommand field with a command that can run the parser (for examples feel free to look at the json files generated by the supported languages). Please note that some languages from the DOTNET family or compiled languages will more than likely require you to compile the program and run the binary using the runCommand field.

## Image output

ArchLens does not support native image outputs of diagrams. Mermaid has a tool called mermaid-cli (mmdc) which does include this feature so if you want a diagram as a png, jpeg or svg you can use mermaid-cli to do this. Once the diagram is saved locally you can use mmdc like this to generate an image of the diagram.

```console
mmdc -i completeView.mmd -o completeView.png
```
