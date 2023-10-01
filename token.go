package sb

import (
	"crypto"
	_ "crypto/md5"
	b64 "encoding/base64"
	"fmt"
	"os"
	"reflect"

	"github.com/NTHU-lsalab/sb/pb"
)

// Privileged returns true if the user is Privileged
func Privileged(RemoteClient bool) bool {
	f, err := os.Open(PrivilegeFile)
	if err != nil || RemoteClient {
		return false
	}
	f.Close()
	return true
}

func Hash(objs ...interface{}) []byte {
	digester := crypto.MD5.New()
	for _, ob := range objs {
		fmt.Fprint(digester, reflect.TypeOf(ob))
		fmt.Fprint(digester, ob)
	}
	return digester.Sum(nil)
}

func HashResult(results []*pb.Result) string {
	str := ""
	for _, result := range results {
		str = str + b64.StdEncoding.EncodeToString(Hash(result))
	}
	return b64.StdEncoding.EncodeToString(Hash(str))
}

func HashSubmission(user string, hw string, result string) string {
	ha := Hash(user, hw, HashValue)
	return b64.StdEncoding.EncodeToString(ha)
}
