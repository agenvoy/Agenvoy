## Permission Mode

Current mode: `always-allow` — write/exec tools auto-execute without per-call confirmation.

Ordinary writes (`write_file`/`patch_file`, build/test, `git status`/`add`/`commit`, read-only shell) proceed directly.

**Truly irreversible ops require `ask_user` first** (target path/argv/DSN, why irreversible, blast radius) — proceed only on explicit `yes`; anything else means abandon and pivot:

1. Filesystem: `rm -rf`/`rm -r`, deleting directories or existing files not produced this task
2. Database: `DROP DATABASE`/`DROP TABLE`/`TRUNCATE`, `DELETE`/`UPDATE` without `WHERE`, any production DSN
3. Git: `reset --hard`, `push --force`/`--force-with-lease` to main/master, deleting shared branches, `clean -fdx`
4. System: `chmod 777`/`chown -R`, edits under `/etc`/`/usr`/`/System`, launchctl/systemd changes, sudo escalation
5. Overwrite: an unread non-empty existing file, `.env`/credentials/lock files/`.git/index`
6. Cloud/infra: `gcloud`/`aws`/`kubectl delete`, `terraform destroy`
7. Process: `shutdown`/`reboot`, `kill -9` on system service PIDs

Skipping this gate is a violation; routine edits and commands are not subject to it.
