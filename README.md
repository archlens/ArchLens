# ArchLens

##
ArchLens is a Python software tool that generates customizable visual package views, showcasing the packages in your system and their dependencies. It offers the flexibility to include or exclude specific packages to suit your requirements for comprehensible views. We are working on making it work also on C#.

Moreover, ArchLens can highlight the differences between your working branch and a specified remote branch, including added or removed dependencies and created or deleted packages, by using green and red highlighting.

Lastly, ArchLens can display the highlighted differences in the system views when a pull request is created on GitHub. It automatically generates the views specified in your config, highlights the differences, and displays them in your pull request, simplifying the review process.

To help you get started, this readme includes various options in combination with the setup of a config file.

## Compatibility

ArchLens is compatible with projects written in **Python versions greater than 3.9** 

## Installation

To install ArchLens, simply use the pip package manager by running the following command:

`pip install archlens-preview` (You need administrative right to perform the operation)

This will download and install the necessary files and dependencies needed for ArchLens to run properly.

## Commands

All commands must be run from the project's root folder

<b>The system has 4 commands:</b>


-`archlens init`- Creates the config template

-`archlens render` - Renders the views specified in the config

-`archlens render-diff` - Renders the differences in the views between your working branch and a specified branch

-`archlens create-action` Creates the github action which will automatically add the difference views to pull requests.

# Using the system

In this section, we will guide you through using the ArchLens system by explaining the commands and output with the example of an API project called 'zeeguu-api' that can be found at https://github.com/zeeguu/api.

Although the project is not large, understanding the system even for this project size of roughly 40 packages can be challenging. To begin generating views, you need to be in the root of your project and run the following command:

- `archlens init`

This will create an "archlens.json" file in your root folder, where you can edit your desired views. This is the initial config:

```json
 {
    "$schema": "https://raw.githubusercontent.com/archlens/ArchLens/master/src/config.schema.json",
    "name": "",
    "rootFolder": "",
    "github": {
        "url": "",
        "branch": "main"
    },
    "saveLocation": "./diagrams/",
    "views": {
        "completeView": {
            "packages": [],
            "ignorePackages": []
        }
    }
}

```


### You can render the views specified in your "archlens.json" file by running the command:
- `archlens render`

This will generate the diagrams for all the views defined in your configuration file and save them in the location specified in the "saveLocation" field of your configuration.
Since currently you only have one view you will only see the following: 

![Zeeguu view](https://raw.githubusercontent.com/archlens/ArchLens/master/.github/readme/zeeguu-api-completeView.png)

## Expanding Packages

If you want to generate another view, in which you want to show the contents of the `core` package for example, you can do it by adding a new view to your definition. E.g. 

```json
 {
    "$schema": "https://raw.githubusercontent.com/archlens/ArchLens/master/src/config.schema.json",
    "name": "",
    "rootFolder": "",
    "github": {
        "url": "",
        "branch": "main"
    },
    "saveLocation": "./diagrams/",
    "views": {
        "completeView": {
            "packages": [],
            "ignorePackages": []
        },
        
 "inside-core": {
      "packages": [
        {
          "path": "core",
          "depth": 1
        }
      ]
    }
}

```

This will generate another view that shows all the dependencies inside the `core` package


![Zeeguu view](https://raw.githubusercontent.com/archlens/ArchLens/master/.github/readme/zeeguu-api-inside-core.png)


## Filtering of packages

The last view we generated above is 	quite dense. One way to solve this problem is to observe that almost all the packages depend on the `core.model` package. When every node depends on it, drawing all the dependencies has little value. We can filter nodes that are shown in the diagram if we use the `ignorePackages` key in a view definition. 

We redefine the view to filter out the `core.mode` package from the view: 

```json

    "inside-core": {
      "packages": [
        {
          "path": "core",
          "depth": 1
        }
      ],
      "ignorePackages": [
        "core.model",
      ]
    },
```

The resulting view is much more relevant for understanding the architecture of this sytem. 

![Zeeguu view](https://raw.githubusercontent.com/archlens/ArchLens/master/.github/readme/zeeguu-api-inside-core-no-model.png)


## Arrows
Each arrow in the system diagram represents a dependency between two packages, and the number on the arrow indicates the number of dependencies going in that direction. If you prefer not to see these arrows, you can use the optional "showDependencyCount" setting, which is a boolean. When set to "false", the dependency count will be hidden in all views. Here is an example of how to set this option in your archlens.json file:

```json
{
    "$schema": "https://raw.githubusercontent.com/archlens/ArchLens/master/src/config.schema.json",
    "name": "zeeguu", # Name of project
    "rootFolder": "zeeguu", # Name of source folder
    "github": {
        "url": "https://github.com/zeeguu/api", # Link to project's Github
        "branch": "master" # Name of main/master branch of project
    },
    "showDependencyCount": false, <------ here we remove the arrows.
    "saveLocation": "./diagrams/", # Location to store generated diagrams
}
```
In this ArchLens config file, the dependency count would be gone. This setting is applied to all of the views.

## Ignore packages
In addition to selecting which packages you want in your diagram, you can also select which packages you want removed from your diagram.

This can be done in two different ways:

```json
"ignorePackages": [
"*test*" #Removes any package which contains the word test
"api/test" #Removes the package api/test and all of its sub packages
]
```

To clarify, the first method using an asterisk (*) will remove any package containing the specified keyword, while the second method will remove only the specified package and all of its sub-packages. This can be useful for cleaning up clutter in the diagram or for excluding certain packages that are not relevant to the analysis.

## The difference views
To generate a difference view using ArchLens, you need to be on a branch other than the one specified in the configuration file. Usually, you would compare your current branch with the main/master branch, but you have the flexibility to choose any branch you desire. For the following example, I have narrowed down the view by filtering out only the "core/model" package.

```json
{
    "$schema": "https://raw.githubusercontent.com/archlens/ArchLens/master/src/config.schema.json",
    "name": "zeeguu",
    "rootFolder": "zeeguu",
    "github": {
        "url": "https://github.com/zeeguu/api",
        "branch": "master"
    },
    "saveLocation": "./diagrams/",
    "views": {
         "coreView":{
            "packages": [
                "core/model" #Looking at core/model, using the path instead of object, because i want to see the entire sub system
            ],
            "ignorePackages": []
        }
    }
}
```
For the next example, the core view is further filtered to show only "core/model". Three changes were made in comparison to the main branch: the package "smart_watch" was deleted, a new package called "smart_watch_two" was added, and a dependency from "word_knowledge" to "model" was removed.

To render this new view displaying the changes, a new command must be run:

- `archlens render-diff`

![Zeeguu core view](https://raw.githubusercontent.com/archlens/ArchLens/master/.github/readme/zeeguu-modelViewdiffView.png)

If there are no diffrences, a diagram without diffrences will still be generated.


## Github action - Pull request

To display the difference views in your pull requests, run the command:

- `archlens create-action`

This command generates the necessary files in the .github folder, creating it if it doesn't already exist. Once this is done, you can create a pull request, and the difference view will be visible to the reviewer, as shown in the image below. If there are no diffrences, a diagram without diffrences will still be generated.

![Zeeguu core view](https://raw.githubusercontent.com/archlens/ArchLens/master/.github/readme/zeeguu-modelViewDiffGithub.png)

## Contributing

Further development on ArchLens is welcomed. To contribute to developing further on ArchLens, we welcome you to fork the repository and propose your additions.

Before you start developing, ensure you have a compatible Python version for running ArchLens. ArchLens have been tested for versions after, including, 3.9, up until, and excluding, version 3.12. There are known issues related to running ArchLens with a version after and including version 3.11.

After ensuring that the current Python version is compatible with ArchLens, we recommend installing the required packages from the files _requirements.txt_ and _dev-requirements.txt_. This will ensure that the necessary packages to run and test your contributions to ArchLens are in your development environment and that they uphold the minimum version requirements. The installing process has been tested using _pip_, and can be done using the following commands:

```
python -m pip install -r requirements.txt
python -m pip install -r dev-requirements.txt
```
(What you use to install pip packages might differ)

Next, to continue setting up your development environment, run the following command:
```
python setup.py develop
```

After following these steps, one can use the commando below to run the commands locally:
```
python ./src/cli_interface.py [cli_command]
```

For an overview of CLI-commands, look [here](https://github.com/archlens/ArchLens/edit/master/README.md#commands).
