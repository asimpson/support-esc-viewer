# support-esc-viewer

This provides a single page which contains the following issue information for support-escalation issues:

- Title
- Labels
- Body
- Comments

This was made because the native Github notification view which only gives you issue title, to view other information you have to click through to the issue detail page.

## Setup

This server assumes you've subscribed to the support-escalations repo in Github.

![image](https://user-images.githubusercontent.com/1048831/192554622-02920078-d8d2-4581-bc75-e75a016a014f.png)

You also need to [create a personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token) with `repo` access.

Set that token as an env variable in your shell like so: `export GITHUB_TOKEN=PASTE_TOKEN_HERE`

The server will fetch all your notifications and then filter by the `support-escalation` repo and fetch additional issue information for each matching issue.
