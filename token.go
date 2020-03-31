package sb

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

const PrivilegeFile = "/etc/judge.priv"
const SecretFile = "/etc/judge.secret"

var secretFmt string

func init() {
	b, err := ioutil.ReadFile(SecretFile)
	if err != nil {
		log.Print("Failed to read secret")
		secretFmt = ""
	}
	secretFmt = string(b)
}

// Privileged returns true if the user is Privileged
func Privileged() bool {
	f, err := os.Open(PrivilegeFile)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func EnsureSecret() {
	if secretFmt == "" {
		panic("empty secret")
	}
}

// WeakTokenForHomework returns a weak token for submission
func WeakTokenForHomework(user, homework string) string {
	msg := fmt.Sprintf(secretFmt, user, homework)
	hash := sha512.Sum512([]byte(msg))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
