package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/NTHU-lsalab/sb"
	"github.com/NTHU-lsalab/sb/pb"
)

func storeSubmission(homework string, submission *pb.StoredSubmission) error {
	b, err := json.Marshal(submission)
	if err != nil {
		return err
	}
	outputFile := filepath.Join(sb.StorageDir, homework, submission.User) + ".json"
	err = ioutil.WriteFile(outputFile+"-", b, 0644)
	if err != nil {
		return err
	}
	return os.Rename(outputFile+"-", outputFile)
}
