from setuptools import setup, find_packages, glob
import os

# Read README for long description
here = os.path.abspath(os.path.dirname(__file__))
try:
    with open(os.path.join(here, "README.md"), encoding="utf-8") as f:
        readme = f.read()
    with open(os.path.join(here, "CHANGELOG.md"), encoding="utf-8") as f:
        changelog = f.read()
    long_description = f"{readme}\n\n{changelog}"
except FileNotFoundError:
    long_description = "See the readme at: https://github.com/archlens/ArchLens/"

setup(
    name="ArchLens",
    version="0.3.0",
    description="Designed for visualizing package dependencies and highlighting differences between"
    " branches in GitHub pull requests. It offers customization options to tailor package views.",
    author="The ArchLens Team",
    author_email="mlun@itu.dk",
    url="https://github.com/archlens/ArchLens",
    packages=find_packages(),
    long_description=long_description,
    long_description_content_type="text/markdown",
    data_files=glob.glob("src/config.**.json"),
    include_package_data=True,
    install_requires=[
        "plantuml",
        "typer",
        "astroid",
        "six",
        "requests",
        "jsonschema",
        "gitpython",
    ],
    classifiers=[
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.9",
    ],
    entry_points={
        "console_scripts": [
            "archlens=src.cli_interface:main",
        ],
    },
)
