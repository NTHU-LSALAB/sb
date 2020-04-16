package sb

import (
	"os"
)

// Privileged returns true if the user is Privileged
func Privileged() bool {
	f, err := os.Open(PrivilegeFile)
	if err != nil {
		return false
	}
	f.Close()
	return true
}
