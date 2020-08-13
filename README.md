# sb
judge &amp; scoreboard server for parallel programming coureses

## Build & Installation

1. git clone this repository.
2. Run `ninja` in the root of this repository. The command builds the `sb` and `xjudge` binaries.
3. Create a scoreboardd user & group `scoreboardd`.
4. Install `xjudge` binary with setgid `scoreboardd`. `sudo install -Dm2711 -gscoreboardd xjudge /usr/local/bin/xjudge`
5. Install the `sb` binary into `scoreboardd`'s home. `sudo install -Dm755 -oscoreboardd -gscoreboardd sb /home/scoreboardd/sb`
6. Create the directory for the scoreboard socket. `sudo install -dm750 -oscoreboardd -gscoreboardd /run/scoreboard`
7. (Optional) Install the TA privilege file `/etc/judge.priv`. Users who can read this file are allowed to use privileged features of the judge. `sudo install -Dm440 -gta /dev/null /etc/judge.priv`

