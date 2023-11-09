from setuptools import setup, find_packages, glob

setup(
    name="ArchLens-Preview",
    version="0.0.7",
    description="Designed for visualizing package dependencies and highlighting differences between branches in GitHub pull requests. It offers customization options to tailor package views.",
    author="The ArchLens Team",
    author_email="sboj@itu.dk",
    url="https://github.com/archlens/ArchLens",
    packages=find_packages(),
    long_description="This is the long description",
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
