import git
import os


def fetch_git_repo(tmp_dir, github_url, branch):
    # Set environment variables to optimize Git operations
    env = os.environ.copy()
    env['GIT_TERMINAL_PROMPT'] = '0'  # Disable prompts
    env['GIT_ASKPASS'] = 'echo'  # Prevent credential prompting

    # Configure clone options for maximum speed
    git_options = [
        '--no-tags',  # Don't fetch any tags
    ]

    # Clone only the specific branch with minimal history and features
    repo = git.Repo.clone_from(
        github_url,
        tmp_dir,
        branch=branch,
        depth=1,
        env=env,
        multi_options=git_options,
        no_checkout=True  # Skip checkout if you don't need files immediately
    )

    # If you do need the files, perform a sparse checkout of just what you need
    repo.git.checkout(branch)

    return repo