import subprocess

# Define the path to the requirements.txt file
requirements_file_path = "requirements.txt"

# Read the packages from the requirements.txt file
with open(requirements_file_path, 'r') as f:
    packages = [line.strip() for line in f if line.strip()]

# Install each package
for package in packages:
    subprocess.call(["pip", "install", package])
