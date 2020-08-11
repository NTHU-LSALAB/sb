#!/bin/python -I

from typing import Optional, List, Union
from dataclasses import dataclass, asdict
import subprocess
import pathlib
import sys
import os
import time
import secrets
import abc
import json
import argparse
import shlex
import tempfile


def tempdir():
    return tempfile.TemporaryDirectory(dir=os.path.expanduser('~'),
                                       prefix='.judge.')


MaybePath = Union[str, pathlib.Path]


@dataclass
class Result:
    passed: bool
    time: float
    verdict: str
    details: Optional[str] = None


class Runner(abc.ABC):
    def __init__(self, *, case: str, exe: str, debug: bool = False):
        self.case = case
        self.exe = exe
        self.debug = debug

    def msg(self, *args):
        if self.debug:
            print(f'[{os.uname().nodename}:{os.getpid()}]',
                  *args,
                  file=sys.stderr)

    @abc.abstractmethod
    def time_limit(self) -> float:
        raise NotImplementedError

    @abc.abstractmethod
    def validate(self, output: pathlib.Path) -> None:
        raise NotImplementedError

    def to_result(self, output: pathlib.Path, returncode: int,
                  walltime: float) -> Result:
        if walltime > self.time_limit():
            return Result(False, walltime, 'time limit exceeded',
                          str(self.time_limit()))
        if returncode != 0:
            return Result(False, walltime, 'runtime error', str(returncode))
        if not os.path.exists(output):
            return Result(False, walltime, 'no output')
        try:
            self.validate(output)
        except ValidationError as e:
            return Result(False, walltime, 'wrong answer', str(e))
        return Result(True, walltime, 'accepted')

    @abc.abstractmethod
    def run(self) -> Result:
        raise NotImplementedError

    @classmethod
    def add_cmdline_options(cls, parser: argparse.ArgumentParser):
        pass

    @classmethod
    def main(cls):
        if os.getegid() != os.getgid():
            gid = os.getgid()
            os.setresgid(gid, gid, gid)

        parser = argparse.ArgumentParser(allow_abbrev=False)
        parser.add_argument('case')
        parser.add_argument('exe')
        parser.add_argument('--debug', action='store_true')
        cls.add_cmdline_options(parser)
        try:
            result: Result = cls(**vars(parser.parse_args())).run()
            print(json.dumps(asdict(result)))
        except KeyboardInterrupt:
            pass


class ValidationError(Exception):
    pass


class SallocRunner(Runner):
    judgein = None
    judgeout = None

    def __init__(self, *, inner=False, **kwargs):
        super().__init__(**kwargs)
        self.is_inner = inner

    @abc.abstractmethod
    def salloc_options(self) -> List[str]:
        raise NotImplementedError

    def srun_options(self) -> List[str]:
        return []

    @abc.abstractmethod
    def exe_args(self) -> List[str]:
        raise NotImplementedError

    def prepare_inner(self):
        pass

    def inner_run(self, judgedir: pathlib.Path) -> Result:
        self.judgeout = judgedir / ('in.' + secrets.token_urlsafe(8))
        self.judgein = judgedir / ('out.' + secrets.token_urlsafe(8))

        self.prepare_inner()

        command = [
            '/usr/bin/srun',
            '--quit-on-interrupt',
            '--disable-status',
            *self.srun_options(),
            self.exe,
            *self.exe_args(),
        ]

        self.msg('running command:',
                 ' '.join(shlex.quote(str(arg)) for arg in command))
        t0 = time.perf_counter()
        try:
            p = subprocess.run(
                command,
                stdin=subprocess.DEVNULL,
                stdout=sys.stderr,
                stderr=sys.stderr,
                timeout=self.time_limit() + 10,
            )
        except subprocess.TimeoutExpired:
            returncode = 65536
        else:
            returncode = p.returncode
        t1 = time.perf_counter()
        walltime = t1 - t0
        if returncode == 65536:
            return Result(False, walltime, 'time limit exceeded+', str(self.time_limit() + 10))
        return self.to_result(self.judgeout, returncode, walltime)

    def inner(self) -> Result:
        with tempdir() as judgedir:
            return self.inner_run(pathlib.Path(judgedir))

    def outer(self) -> Result:
        command = [
            '/usr/bin/salloc',
            f'-J{self.case}',
            '--quiet',
            *self.salloc_options(),
            '/usr/bin/python',
            '-I',
            sys.argv[0],
            self.case,
            self.exe,
            '--inner',
        ]
        if self.debug:
            command.append('--debug')

        self.msg('running command:',
                 ' '.join(shlex.quote(str(arg)) for arg in command))
        t0 = time.perf_counter()
        p = subprocess.run(
            command,
            stdout=subprocess.PIPE,
        )
        t1 = time.perf_counter()

        if p.returncode != 0:
            return Result(False, t1 - t0, 'internal error',
                          f'inner non zero exit status: {p.returncode}')
        try:
            data = json.loads(p.stdout)
        except json.JSONDecodeError as e:
            return Result(False, t1 - t0, 'internal error',
                          f'cannot decode inner output: {e}')
        try:
            return Result(**data)
        except ValueError:
            return Result(False, t1 - t0, 'internal error',
                          'malformed inner result')

    def run(self):
        if self.is_inner:
            return self.inner()
        else:
            return self.outer()

    @classmethod
    def add_cmdline_options(cls, parser):
        parser.add_argument('--inner', action='store_true')


class HW2Runner(SallocRunner):
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.options = {}
        with open(f'/home/ipc20/ta/hw2/hidden/{self.case}') as file:
            for line in file:
                k, v = line.strip().split('=')
                self.options[k] = str(v)

    def salloc_options(self):
        return ['-N{}'.format(self.options['N']), '-n{}'.format(self.options['n']), '-c{}'.format(self.options['c'])]

    def srun_options(self):
        if self.debug:
            return []
        return ['-onone']

    def time_limit(self):
        return int(self.options['timelimit'])

    def prepare_inner(self):
        pass

    def exe_args(self):
        return [self.options['c'],
                *self.options['pos'].split(), 
                *self.options['tarpos'].split(),
                self.options['width'], 
                self.options['height'], 
                self.judgeout]

    def validate(self, output_file):
        valid = self.options['valid']
        ret = subprocess.run(['/home/ipc20/ta/hw2/hw2-diff', valid, self.judgeout], stdout=subprocess.DEVNULL)
        if ret.returncode != 0:
            raise ValidationError(f'the correctness ratio is below 95%')


if __name__ == '__main__':
    HW2Runner.main()
