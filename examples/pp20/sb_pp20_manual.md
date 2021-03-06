# PP20 Scoreboard Manual
Update 2020.10.19

# Installation
Install go dependence
```sh
go get -u github.com/golang/protobuf/protoc-gen-go
mv ~/go ~/.pkg/
```

```sh
git clone https://github.com/NTHU-LSALAB/sb.git
cd sb
export PATH=$PATH:$HOME/.pkg/go/bin
ninja
```

## Testcases
* Put testcases under `/home/pp20/share/XXX/testcases`.
* Each testcase should specified `salloc` options value (`N`, `n`, `c` and so on) and `argc` values to be feed to the program.
* The testcase can be any filetype you like (e.g. `.txt`, `.json`). Only need to implement the corresponding parser in `runner.py`
* Make sure normal user can't write to testcases.

## Makefile
* Put sample `Makefile` under `/home/pp20/share/XXX/sample`.

## Create Config file
* Login to user `scoreboardd`
* Create a file `XXX.toml` under `/home/scoreboardd/pp20/config`
    ```toml
    # The ninja build target of the homework (or the name of the executable)
    # If not provided, defaults to the name of the homework (which is the name of the .toml file)
    target = "XXX"

    # The absolute path runner for the homework
    runner = "/home/pp20/ta/XXX/runner.py"

    # Files to copy before compiling
    files = [
        {name = "XXX.cc"},
        {name = "Makefile", fallback = "/home/pp20/share/XXX/sample/Makefile"},
    ]

    # Time penalty for failing a test case
    penalty_time = 60

    # Name of the test cases
    cases = [
        "[XX-XX].txt"
    ]
    ```

## `runner.py`
* Create a file named `runner.py` under `/home/pp20/ta/XXX`
* Edit the class `XXXRunner`
    ```py
    class XXXRunner(SallocRunner):
        def __init__(self, **kwargs):
            super().__init__(**kwargs)
            self.options = {}
            # [modified this] parse and load the testcase config here
            with open(f'/home/pp20/share/XXX/testcases/{self.case}') as file:
                for line in file:
                    k, _, v = line.partition('=')
                    self.options[k] = int(v)

        # [modified this] define the salloc options
        def salloc_options(self):
            return ['-N1', '-n{}'.format(self.options['nproc']), '-c1']

        # [modified this] define srun options here
        def srun_options(self):
            if self.debug:
                return [f'-o{self.judgeout}', '-eall']
            return [f'-o{self.judgeout}', '-enone']

        def time_limit(self):
            return self.options['timelimit']

        def prepare_inner(self):
            pass

        # [modified this] args to be passed to the program
        def exe_args(self):
            return [str(self.options['r']), str(self.options['k'])]

        # [modified this] check the answer
        def validate(self, output_file):
            answer = str(self.options['answer'])
            with open(output_file) as file:
                output = file.read().rstrip()
            if output != answer:
                raise ValidationError(f'expected {answer!r}; got {output!r}')
    ```
* `chmod 775 runner.py`

## Restart the scoreboard
* Login to user `scoreboardd`
* run `squeue` to make sure no one is running the judge right now
* `tmux a`
* `ctrl-C` the current running scoreboard.
* Make sure you are under the diectory `/home/scoreboardd/pp20`
* `./sb --outputdir /srv/http/pp20/scoreboard/`

## Testing
To test the scoreboard, do the following.
* Under the same directory as the source code, run `xjudge --homework XXX`
* The result will be see on the webpage apollo.cs.nthu.edu.tw/pp20/scoreboard/XXX

## Release the judge
```sh
cd /usr/local/bin
sudo ln -s xjudge XXX-judge
```