import git


def fetch_git_repo(tmp_dir, github_url, branch):

    repo_path = github_url
    repo = git.Repo.clone_from(repo_path, tmp_dir)

    original_branch = repo.active_branch.name

    repo.git.checkout(branch)
    repo.remotes.origin.pull()

    repo.git.checkout(original_branch)
