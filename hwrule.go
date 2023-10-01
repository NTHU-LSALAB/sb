package sb

import (
	"path/filepath"

	"github.com/NTHU-lsalab/sb/intrange"
	"github.com/NTHU-lsalab/sb/pb"

	"github.com/BurntSushi/toml"
)

func LoadHomework(filename string) *pb.Homework {
	hw := new(struct {
		Target       string
		Runner       string
		RemoteRunner string
		Files        []*pb.SourceFile
		PenaltyTime  toml.Primitive `toml:"penalty_time"`
		Cases        []string
	})
	metadata, err := toml.DecodeFile(filename, hw)
	if err != nil {
		panic(err)
	}
	name := filepath.Base(filename)
	name = name[:len(name)-len(filepath.Ext(name))]
	if hw.Target == "" {
		hw.Target = name
	}
	var penaltyTime float64
	err = metadata.PrimitiveDecode(hw.PenaltyTime, &penaltyTime)
	if err != nil {
		var penaltyTimeAsInt int64
		err = metadata.PrimitiveDecode(hw.PenaltyTime, &penaltyTimeAsInt)
		if err != nil {
			panic(err)
		}
		penaltyTime = float64(penaltyTimeAsInt)
	}
	var expandedCases []string
	for _, casestr := range hw.Cases {
		expandedCases = append(expandedCases, intrange.MustExpand(casestr)...)
	}
	hw.Cases = expandedCases
	return &pb.Homework{
		Name:         name,
		Target:       hw.Target,
		Runner:       hw.Runner,
		RemoteRunner: hw.RemoteRunner,
		Files:        hw.Files,
		PenaltyTime:  penaltyTime,
		Cases:        hw.Cases,
	}
}
