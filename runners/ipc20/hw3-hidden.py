#!/home/ipc20/ta/python382/bin/python3.8 -I

from typing import Optional, List, Union
from dataclasses import dataclass, asdict
import subprocess
import pathlib
import shutil
import sys
import enum
import os
import time
import secrets
import abc
import json
import argparse
import shlex
import tempfile
import ctypes

libpngdiff = ctypes.CDLL('/home/ipc20/ta/hw3/libpngdiff.so')
libpngdiff.pngdiff.argtypes = (
    ctypes.c_char_p, ctypes.c_char_p, ctypes.POINTER(ctypes.c_char_p), ctypes.POINTER(ctypes.c_double)
)
libpngdiff.pngdiff.restype = ctypes.c_int
libpngdiff.freestr.argtypes = (ctypes.c_char_p, )


def tempdir():
    return tempfile.TemporaryDirectory(
        dir=os.path.expanduser('~'), prefix='.judge.'
    )


MaybePath = Union[str, pathlib.Path]


class Verdict(str, enum.Enum):
    AC = 'accepted'
    WA = 'wrong answer'
    RE = 'runtime error'
    TLE = 'time limit exceeded'
    TLE_PLUS = 'time limit exceeded+'
    IE = 'internal error'


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
            print(
                f'[{os.uname().nodename}:{os.getpid()}]',
                *args,
                file=sys.stderr
            )

    def print_command(self, command):
        self.msg(
            'running command:',
            ' '.join(shlex.quote(str(arg)) for arg in command),
        )

    @abc.abstractmethod
    def time_limit(self) -> float:
        raise NotImplementedError

    @abc.abstractmethod
    def validate(self, output: pathlib.Path, time: float) -> Result:
        raise NotImplementedError

    def to_result(
            self, output: pathlib.Path, returncode: int, walltime: float
    ) -> Result:
        if walltime > self.time_limit():
            return Result(False, walltime, Verdict.TLE, str(self.time_limit()))
        if returncode != 0:
            return Result(False, walltime, Verdict.RE, str(returncode))
        if not os.path.exists(output):
            return Result(False, walltime, Verdict.WA, 'no output')
        return self.validate(output, walltime)

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

        self.print_command(command)
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
            return Result(
                False, walltime, Verdict.TLE_PLUS, str(self.time_limit() + 10)
            )
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
            sys.executable,
            '-I',
            sys.argv[0],
            self.case,
            self.exe,
            '--inner',
        ]
        if self.debug:
            command.append('--debug')

        self.print_command(command)
        t0 = time.perf_counter()
        p = subprocess.run(
            command,
            stdout=subprocess.PIPE,
        )
        t1 = time.perf_counter()

        if p.returncode != 0:
            return Result(
                False, t1 - t0, Verdict.IE,
                f'inner non zero exit status: {p.returncode}'
            )
        try:
            data = json.loads(p.stdout)
        except json.JSONDecodeError as e:
            return Result(
                False, t1 - t0, Verdict.IE, f'cannot decode inner output: {e}'
            )
        try:
            return Result(**data)
        except ValueError:
            return Result(False, t1 - t0, Verdict.IE, 'malformed inner result')

    def run(self):
        if self.is_inner:
            return self.inner()
        return self.outer()

    @classmethod
    def add_cmdline_options(cls, parser):
        parser.add_argument('--inner', action='store_true')


class SbatchRunner(Runner):
    judgeout = None
    judgein = None

    def __init__(self, *, inner=False, **kwargs):
        super().__init__(**kwargs)
        self.is_inner = inner

    @abc.abstractmethod
    def sbatch_options(self) -> List[str]:
        raise NotImplementedError

    @abc.abstractmethod
    def exe_args(self) -> List[str]:
        raise NotImplementedError

    def prepare_inner(self, judgedir: pathlib.Path):
        pass

    def inner_run(self, judgedir: pathlib.Path) -> Result:
        self.judgeout = judgedir / ('in.' + secrets.token_urlsafe(8))
        self.judgein = judgedir / ('out.' + secrets.token_urlsafe(8))

        self.prepare_inner()

        command = [
            self.exe,
            *self.exe_args(),
        ]

        self.print_command(command)
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
            return Result(
                False,
                walltime,
                Verdict.TLE_PLUS,
                str(self.time_limit() + 10),
            )
        return self.to_result(self.judgeout, returncode, walltime)

    def inner(self) -> Result:
        with tempfile.TemporaryDirectory(dir='/dev/shm',
                                         prefix='.judge.') as judgedir:
            return self.inner_run(pathlib.Path(judgedir))

    def outer(self) -> Result:
        with tempdir() as judgedir:
            judgedir = pathlib.Path(judgedir)
            inner_stdout = judgedir / ('stdout.' + secrets.token_urlsafe(8))
            inner_stderr = judgedir / ('stderr.' + secrets.token_urlsafe(8))

            command = [
                '/usr/bin/sbatch',
                f'-J{self.case}',
                '--quiet',
                '--wait',
                f'--output={inner_stdout}',
                f'--error={inner_stderr}',
                *self.sbatch_options(),
                sys.argv[0],
                self.case,
                self.exe,
                '--inner',
            ]
            if self.debug:
                command.append('--debug')

            self.print_command(command)
            t0 = time.perf_counter()
            p = subprocess.run(command, stdout=sys.stderr)
            t1 = time.perf_counter()

            if self.debug:
                try:
                    stderrf = open(inner_stderr, 'rb')
                except FileNotFoundError:
                    pass
                else:
                    with stderrf:
                        shutil.copyfileobj(stderrf, sys.stdout.buffer)
            if p.returncode != 0:
                return Result(
                    False, t1 - t0, Verdict.IE,
                    f'inner non zero exit status: {p.returncode}'
                )
            try:
                with open(inner_stdout) as stdoutf:
                    data = json.load(stdoutf)
            except FileNotFoundError:
                return Result(
                    False, t1 - t0, Verdict.IE, f'inner output not found'
                )
            except json.JSONDecodeError as e:
                return Result(
                    False, t1 - t0, Verdict.IE,
                    f'cannot decode inner output: {e}'
                )
        try:
            return Result(**data)
        except ValueError:
            return Result(False, t1 - t0, Verdict.IE, 'malformed inner result')

    def run(self):
        if self.is_inner:
            return self.inner()
        return self.outer()

    @classmethod
    def add_cmdline_options(cls, parser):
        parser.add_argument('--inner', action='store_true')


class Lab2Runner(SallocRunner):
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.options = {}
        with open(f'/home/ipc20/ta/lab2/testcases/{self.case}') as file:
            for line in file:
                k, _, v = line.partition('=')
                self.options[k] = int(v)

    def salloc_options(self):
        return ['-N1', '-n{}'.format(self.options['nproc']), '-c1']

    def srun_options(self):
        if self.debug:
            return [f'-o{self.judgeout}', '-eall']
        return [f'-o{self.judgeout}', '-enone']

    def time_limit(self):
        return self.options['timelimit']

    def prepare_inner(self):
        pass

    def exe_args(self):
        return [str(self.options['r']), str(self.options['k'])]

    def validate(self, output_file, time):
        answer = str(self.options['answer'])
        with open(output_file) as file:
            output = file.read().rstrip()
        if output != answer:
            return Result(
                False, time, Verdict.WA, f'expected {answer!r}; got {output!r}'
            )
        return Result(True, time, Verdict.AC)


class HW3Runner(SbatchRunner):
    def __init__(self, **kwargs):
        super().__init__(**kwargs)

    def sbatch_options(self):
        return ['-ppp', '-N1', '-n1', '-c1', '--gres=gpu:1']

    def srun_options(self):
        if self.debug:
            return [f'-o{self.judgeout}', '-eall']
        return [f'-o{self.judgeout}', '-enone']

    def time_limit(self):
        return 30

    def prepare_inner(self):
        shutil.copy(f'/home/ipc20/ta/hw3/hidden/{self.case}.png', self.judgein)

    def exe_args(self):
        return [self.judgein, self.judgeout]

    def validate(self, output_file, time):
        answer = f'/home/ipc20/ta/hw3/hidden/{self.case}.out.png'
        cstr = ctypes.c_char_p()
        percent = ctypes.c_double()
        retval = libpngdiff.pngdiff(
            answer.encode(), str(self.judgeout).encode(),
            ctypes.byref(cstr), ctypes.byref(percent),
        )
        msg = cstr.value.decode()
        libpngdiff.freestr(cstr)
        if retval == 66:
            return Result(False, time, Verdict.WA, msg)
        if retval:
            return Result(False, time, Verdict.IE, str(percent))
        return Result(True, time, Verdict.AC, msg)


if __name__ == '__main__':
    HW3Runner.main()
