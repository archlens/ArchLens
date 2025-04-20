import git

def fetch_git_repo(tmp_dir, github_url, branch):
    repo = git.Repo.clone_from(
        github_url,
        tmp_dir,
        branch=branch,  # Clone directly to the desired branch
        depth=1  # Shallow clone (only latest commit)
    )

    return repo