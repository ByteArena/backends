# dotgit

ssh-wrapper and ssh-keystore for GIT repositories.

Expects a json config file in `/etc/dotgit.conf`, like so:

```json
{
    "DatabaseURI":"http://host/graphql",
    "GitRepositoriesPath":"/home/git/repositories"
}
```