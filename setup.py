from setuptools import setup, find_packages

setup(
    name="MT-diagrams",
    version="0.0.6",
    description="Thesis project",
    author="Nikolai Perlt",
    author_email="npe@itu.dk",
    url="https://github.com/Perlten/MT-diagrams",
    packages=find_packages(),
    long_description="This is the long description",
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
            "mt-diagrams=src.cli_interface:main",
        ],
    },
)
