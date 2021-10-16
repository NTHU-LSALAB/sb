# Runners in IPC20

The interface between `xjudge` and runners is that xjudge calls the runner for each test case. And the runners return a JSON in stdout.
`xjudge` handles the compilation and the concurrent execution of input test cases, while runners actual runs the code with SLURM and check the correctness.
In fact, `xjudge` doesn't even care that the runners here uses SLURM to run students' code.

The runners here are python scripts. They can be any script with a proper shebang (`#!`) that conforms to the interface stated above.

There are two types of runners: SallocRunner and SbatchRunner:

* SallocRunner: `xjudge` -> runner (outer) -> salloc -> runner (inner) -> srun `a.out` (code to test for correctness)
* SbatchRunner: `xjudge` -> runner (outer) -> sbatch -> runner (inner) -> srun `a.out`

The runners call themselves with either salloc and sbatch.
The first call to the runner (outer) set SLURM parameters and then calls itself with either salloc and sbatch.
The second call to the runner (inner) from itself times, runs the executable `a.out`, and validates the output.
The inner/outer split ensures that when `a.out` is actuall run by the inner runner, SLRUM resources are already allocated.
With resources immediately available, when timing the execution we don't account queuing time.

The practical differences between SallocRunner and SbatchRunner are:

* SallocRunner's inner runner runs on the head node. SbatchRunner's inner runner runs on the compute node.
  Therefore with SbatchRunner it is possible to have the output file live on the compute node's memory.
  With SallocRunner, instead, the output must be on NFS.

* SallocRunner is much faster than SbatchRunner with small (fast) test cases.
  This is due to `sbatch --wait` taking much more time to complete after the job exits than `salloc`.
  The wait time difference just makes users wait longer for the outer runner and shouldn't affect timing which is done by the inner runner.
