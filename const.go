package sb

// DefaultAddr is the default address the scoreboard server is listening to
const DefaultAddr = "140.114.91.183:7122"
const SSHPort = 22

// PrivilegeFile is a file that grants access to the judge's privileged options for
// users who have read access to it
const PrivilegeFile = "/etc/judge.priv"

// StorageDir is the directory the scoreboard server stores submissions
const StorageDir = "storage"

// Only username begins with `StudentPrefix` will be ranked.
const StudentPrefix = "[SC_CAMP]"

// Prefix for remote client
const RemotePrefix = "[SC_CAMP] "

const HashValue = "z/7ZEhQ[{gF9!Q,GfwDBiHYR{@V{CU%xd+7S/4kdHL8!V37M?742-*EC.@cWC9d4X@uDnu__DZA-vm-m!8t9m*{&vq-BRPA5H&iG+3FhV]FwaBU7PB!;h5HRp!-i/D#easdjfkasdjlf"

// Number of parallel runner.
const ParallelRunners_SLURM = 4