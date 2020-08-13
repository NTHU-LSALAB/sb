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

## Running the Scoreboard

Run the `sb` binary as the scoreboard user. The `sb` command runs the scoreboard server, which accepts judge requestse from the `xjudge` command and outputs the scoreboard as HTML files.

* Configuration files are read from `./config`.
* Data is stored in `./storage`
* HTML scoreboard is output in the `./out` directory. This can be changed by the `--outputdir` flag.

## Judging Procedure

1. Every time `xjudge` is invoked by a user, it first determines which homework it is judging. Running `xjudge --homework hw1` judges `hw1`. If `/usr/local/bin/hw1-judge is a symbolic link to xjudge`, then running `hw1-judge` also judges `hw1`.
2. It communicates with the scoreboard server to ask about the configuration of the the homework. See [Configuration](#configuration).
3. It copies the *files* to a temporary directory.
4. It tries to build the *target* using `ninja`.
5. It run the *cases* with the *runner*. See [Runner](#runner).
6. After collecting the results from the runners, the judge submit the results to the scoreboard.

## Homework Configuration

### Configuration

Homeworks are specified in the scoreboard's `./config/*.toml` files. See `configs/` in this repository for examples. Each homework is specified by a config file. Each config file specify:
1. `target`: the ninja build target.
2. `runner`: the absolute path of the runner.
3. `files`: mandantory and optional files for the homework. 
4. `penalty_time`: time penalty for failing a test case in seconds.
5. `cases`: test case names.

### Runner

Runners are used to actually run each test case, specified by the test cases' name.

`xjudge` runs each test case with: `runner [--debug] casename executable`. Where:
   * `runner` is the absolute path of the runner
   * `--debug` is a optional flag used to enable verbose output of the runner
   * `casename` is the name of the test case
   * `executable` is the *target* executable built by the students' code
   For each case, the runner outputs JSON in stdout, with 4 attributes:
   * `passed`: bool, whether the test case is passed
   * `time`: float, the execution time of the test case
   * `verdict`: string, such as `Accepted`, `Wrong Answer`, etc
   * `details`: string, optional description for the verdict
