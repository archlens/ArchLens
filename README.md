# MT-Diagrams

MT-diagrams is a software tool designed for Python systems that enables you to create visual package views. These views display the various packages included in your system and their dependencies. With MT-diagrams, you have the flexibility to include or exclude specific packages based on your requirements.

For illustration purposes, we'll use the [Zeeguu](https://github.com/zeeguu/api) GitHub project as an example. The following diagram highlights the core packages included in the project.

Note: When running the tool for the first time in a while, it might take a while to generate a diagram, due to our plantuml server currently being hosted on a free plan. Once the service is awake, the speed increases.

![Zeeguu core view](.github/readme/zeeguu-coreView.png)

In addition, the system can identify and highlight the differences between your working branch and a specified remote branch, including added or removed dependencies, as well as created or deleted packages.

To demonstrate this functionality, take a look at the following view that illustrates the differences between a development branch and the main branch of zeegu, on the development branch we can see that some dependencies have been added (green arrows), and some deleted (red arrows).

![Zeeguu diff view](.github/readme/zeeguu-diffview.png)

## Installation

To install mt-diagrams, simply use the pip package manager by running the following command:

`pip install mt-diagrams` (You might need administrative right to perform the operation)

This will download and install the necessary files and dependencies needed for mt-diagrams to run properly.

## Setup

To get started run the command `mt-diagrams create-config` this will create a basic config file defining a basic view that will showcase all packages included in the system

Below you can see the basic config file created

```json
{
    "$schema": "https://raw.githubusercontent.com/Perlten/MT-diagrams/master/config.schema.json",
    "name": "",
    "rootFolder": "",
    "github": {
        "url": "",
        "branch": "main"
    },
    "saveLocation": "./diagrams/",
    "showDependencyCount": true,
    "packageColor": "Azure",
    "views": {
        "completeView": {
            "packages": [],
            "ignorePackages": [],
            "usePackagePathAsLabel": true
        }
    }
}
```

For mt-diagrams to work you will need to fill the fields `name` and `rootFolder`, name being the name of your project, and rootFolder the folder containing your source code, starting from your project root.

- `name`: This value will be prefixed to all your diagrams.
- `rootFolder`: This points to the folder containing the root packages in your system (usually named src or similar).
- `github`: Field in the configuration file allows you to specify a remote GitHub repository and its branch to facilitate difference views
    - `url`: The url of the repository
    - `branch`: The branch for comparison
- `saveLocation`: This is the folder where created views will be saved.
- `showDependencyCount`: This is a boolean, if set to True the dependency arrows in the diagram will show how many imports there are between the packages.
- `packageColor`: Color of packages, you can choose from "GoldenRod" or "Azure".
- `views`: This contains a map with view names as keys and the following sub-fields: 
    - `packages`: This specifies the packages to include in the project. Each path will start at the root_folder, and any path given must exist in the project. If you provide the path "api/test", it means that you want to include the rootFolder/api/test in the graph.
    When entering paths in packages, you are telling the diagram that you only want to include those packages and their sub-packages and dependencies.
    Alternatively to providing a path which includes a package and its entire sub-domain, you can give the following object instead
    ```json
    "packages": [
        {
        "packagePath": "api/test",
        "depth": 2
        },
        "core/controller"
    ]
    ```
    This example will add "rootFolder/api/test" + the 2 layers below it to the diagram, aswell as "core/controller" and its sub-domain and show how all of those packages relate to eachother.

    You provide this as a path to the folder from the rootFolder (e.g., "api.server" will point to a sub-package in the api package named server).
    - `IgnorePackages`:
    There are three different ways to ignore packages:

    Specify by name: This will exclude any package with a given name.
    Specify path to package: This will exclude a package at a given path.
    Specify a name with an asterisk: This will remove any package with the name in its path.
    Here is an example of how to use ignorePackages:

    Example:
    ```
    ignorePackages: [

        - "api/car",  // This removes the package at the path "api/car" from the diagram
        - "scooter",  // This will remove any package with the name "scooter" from the diagram
        - "*test*"     // This will remove any package with the name "test" in its path from the diagram
    
    ]
    ```
    

    - `usePackagePathAsLabel` (Optional: Set to true  by default):
    If usePackagePathAsLabel is set to false, the package name and the end of a path will be the names in the diagram. For example, api/car will have a module named "api" and one named "car".

    If usePackagePathAsLabel is set to true, the paths will be displayed instead. This would result in the packages being named: "api", "api/car".
    

## CLI

The CLI tool has four available commands:

- `mt-diagrams --help`: This command provides instructions on how to use the tool and its available commands.

- `mt-diagrams render`: This command generates a package diagram based on the configuration file. This is the main command used to generate diagrams based on the provided configuration file.

- `mt-diagrams render-diff`: This command generates a package diagram highlighting differences between the working branch and the specified branch in the config file. This command is useful for comparing package dependencies between different branches in a project.

- `mt-diagrams create-config`: This command generates a basic configuration file defining a view that showcases all packages included in the system. This command is useful for quickly generating a configuration file to get started with the tool. This command should be run in the root of your project

- `mt-diagrams create-action`: This creates all of the necessary files for diagrams to be generated when creating a pull request. When creating a pull request, the branch you're working on will display the differences in comparison to the branch specified in the github["branch"] in the above config.
