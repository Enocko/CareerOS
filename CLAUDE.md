# CareerOS

## Working style

Work autonomously within the task you are given. Proceed with routine file edits, debugging, tests, builds, and reasonable implementation decisions without asking whether to continue. When details are minor, choose a sensible default and proceed.

Ask before destructive or difficult-to-reverse actions, deleting important data, changing production systems, deploying or publishing, pushing or merging changes to shared branches, spending money, exposing secrets, or sending messages to others.

Also ask when a missing requirement would materially change the result. Otherwise, finish the task, verify your changes, and summarize what you changed and any remaining issues.

## Notes

- Pushing to `main` deploys to Render, so it counts as deploying: ask first.
- The repo lives in an iCloud-synced Desktop folder. After `git stash` or `git checkout`, check for `"<name> 2.<ext>"` duplicate files (`find . -name "* 2*" -not -path "*/node_modules/*"`). A duplicate migration breaks golang-migrate.
- Database-backed tests need Postgres (`make up`, which needs Docker running). Run them with `CI=true` so a missing database fails instead of skipping.
