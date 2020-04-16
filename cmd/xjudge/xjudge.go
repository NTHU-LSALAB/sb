package main

import (
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/NTHU-lsalab/sb"
	"github.com/NTHU-lsalab/sb/judge"

	"github.com/spf13/pflag"
)

const configRoot = "/usr/local/etc/judge.d"

func parseOptions() *judge.Options {
	opt := &judge.Options{}

	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("Failed to determine current user: %v", err)
	}

	fs := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	fs.SortFlags = false

	fs.StringVarP(&opt.Chdir, "chdir", "C", "", "Change the directory before judging")
	fs.StringVar(&opt.AsUser, "as", currentUser.Username, "Run the judge as the user. Privileged option.")
	fs.StringVar(&opt.RuleFile, "rule", "", "Run the judge with the given rule file. Privileged option.")
	homework := "<unknown>"
	if strings.HasSuffix(os.Args[0], "-judge") {
		homework = filepath.Base(os.Args[0])
		homework = homework[:len(homework)-6]
	}
	fs.StringVar(&opt.Homework, "homework", homework, "Judge the specific homework.")
	fs.StringVar(&opt.Bin, "bin", "", "Skip compiling and use the given binary. Privileged option.")
	fs.StringVar(&opt.Server, "server", sb.DefaultAddr, "Address of the scoreboard server. If it contains a slash, it is treated as a unix domain socket, otherwise it is treated as a tcp socket.")
	fs.IntVar(&opt.MedianOf, "median-of", 1, "Run each case multiple times and pick the median. Must be an odd integer.")

	fs.StringArrayVarP(&opt.ExcludeCases, "exclude", "x", nil, "Exclude the given test cases. Specify this option multiple times to exclude multiple test cases.")
	fs.StringArrayVarP(&opt.IncludeCases, "include", "i", nil, "Include the given test cases. Specify this option multiple times to include multiple test cases. --include takes higher priority than exclude. If --include is specified but --exclude is not specified, the judge will only run only the --include'd test cases. For both --include and --exclude, []-expansion is supported. --include=case[01-03] expands to --include=case01 --include=case02 --include=case03. --exclude=case[01,04] expands to --exclude=case01 --exclude=case04.")

	fs.Parse(os.Args[1:])

	return opt
}

func main() {
	options := parseOptions()
	judge.MainOptions(options)
}
