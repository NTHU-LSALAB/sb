package sb

// DefaultAddr is the default address the scoreboard server is listening to
const DefaultAddr = "/run/scoreboard/sb.sock"

// PrivilegeFile is a file that grants access to the judge's privileged options for
// users who have read access to it
const PrivilegeFile = "/etc/judge.priv"

// StorageDir is the directory the scoreboard server stores submissions
const StorageDir = "storage"
