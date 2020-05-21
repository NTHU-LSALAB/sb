package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/NTHU-lsalab/sb"
	"github.com/NTHU-lsalab/sb/colors"
	"github.com/NTHU-lsalab/sb/pb"

	"google.golang.org/grpc"
)

var username string
var home string

func init() {
	log.SetFlags(0)
	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("cannot get user: %v", err)
	}
	username = currentUser.Username
	home = currentUser.HomeDir
}

func tempdir() string {
	dir, err := ioutil.TempDir(home, ".judge.*")
	if err != nil {
		log.Fatalf("failed to create temporary directory: %v", err)
	}
	return dir
}

type red string

func (r red) String() string {
	return "\x1b[31m" + string(r) + "\x1b[0m"
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		panic(err)
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		panic(err)
	}
	return nil
}

func lookForCopy(ctx context.Context, filename, fallback, targetdir string) bool {
	err := copyFile(filename, filepath.Join(targetdir, filename))
	if err == nil {
		log.Printf("Looking for %s: %s\n", filename, colors.Green("OK"))
		return true
	} else if fallback == "" {
		log.Printf("Looking for %s: %s\n", filename, colors.Red("Not Found"))
		return false
	} else {
		log.Printf("Looking for %s: %s\n", filename, colors.Yellow("Not Found"))
		err = copyFile(fallback, filepath.Join(targetdir, filename))
		if err == nil {
			log.Printf("Using fallback: %s: %s\n", fallback, colors.Green("OK"))
			return true
		}
		log.Printf("Using fallback: %s: %s\n", fallback, colors.Red("Failed"))
		return false
	}
}

func printCommand(command []string) {
	args := []interface{}{"Running:"}
	for _, part := range command {
		args = append(args, part)
	}
	log.Println(args...)
}

func compile(ctx context.Context, rule Rule, dir string) bool {
	for _, filename := range rule.Mandantory {
		if ctx.Err() != nil {
			return false
		}
		if !lookForCopy(ctx, filename, "", dir) {
			return false
		}
	}
	for _, pair := range rule.Optional {
		if ctx.Err() != nil {
			return false
		}
		if !lookForCopy(ctx, pair.Name, pair.Fallback, dir) {
			return false
		}
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/ninja", "-C", dir, rule.Target)
	printCommand(cmd.Args)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Printf("Cannot compile executable\n")
		return false
	}
	_, err = os.Stat(filepath.Join(dir, rule.Target))
	if err != nil {
		log.Printf("Compilation succeed but executable wasn't generated\n")
		return false
	}
	// TODO: check that the output is executable
	return true
}

// OptionalFile specifies the name of an optional file and the fallback file
// if the file is not found
type OptionalFile struct {
	Name     string
	Fallback string
}

// Rule defines a rule of a given homework
type Rule struct {
	Target      string
	Mandantory  []string
	Optional    []OptionalFile
	Runner      string
	SkipCompile bool
	MedianOf    int
	Debug       bool
}

// judgeRequest is a request for judgeing a single case
type judgeRequest struct {
	CaseID     int
	CaseName   string
	Executable string
	Runner     string
	Debug      bool
}

// judgeResult is the result of judging a single case
type judgeResult struct {
	CaseID   int    `json:"-"`
	CaseName string `json:"-"`
	Passed   bool
	Time     float64
	Verdict  string
	Details  string
}

func (jr judgeResult) toProto() *pb.Result {
	return &pb.Result{
		Case:    jr.CaseName,
		Passed:  jr.Passed,
		Time:    jr.Time,
		Verdict: jr.Verdict,
	}
}

func (jr judgeResult) formatDescription() string {
	var verdict string
	if jr.Passed {
		verdict = colors.Green(jr.Verdict)
	} else {
		verdict = colors.Red(jr.Verdict)
	}
	if jr.Details == "" {
		return verdict
	}
	return verdict + ": " + jr.Details
}

func judgeCase(ctx context.Context, jr judgeRequest) judgeResult {
	var cmd *exec.Cmd
	if jr.Debug {
		cmd = exec.Command(jr.Runner, "--debug", jr.CaseName, jr.Executable)
	} else {
		cmd = exec.Command(jr.Runner, jr.CaseName, jr.Executable)
	}
	output := bytes.NewBuffer(nil)
	cmd.Stdout = output
	// cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	t0 := time.Now()
	err := cmd.Start()
	if err != nil {
		return judgeResult{
			jr.CaseID,
			jr.CaseName,
			false,
			time.Now().Sub(t0).Seconds(),
			"internal error",
			fmt.Sprintf("could not start runner: %v", err),
		}
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		log.Println("Getpgid failed")
		pgid = cmd.Process.Pid
	}
	cmdDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			syscall.Kill(-pgid, syscall.SIGTERM)
			select {
			case <-time.After(time.Second * 5):
				syscall.Kill(-pgid, syscall.SIGKILL)
			case <-cmdDone:
			}
		case <-cmdDone:
		}
	}()
	err = cmd.Wait()
	close(cmdDone)
	if err != nil {
		return judgeResult{
			jr.CaseID,
			jr.CaseName,
			false,
			time.Now().Sub(t0).Seconds(),
			"internal error",
			fmt.Sprintf("could not execute runner: %v", err),
		}
	}
	result := judgeResult{
		CaseID:   jr.CaseID,
		CaseName: jr.CaseName,
	}
	err = json.Unmarshal(output.Bytes(), &result)
	if err != nil {
		return judgeResult{
			jr.CaseID,
			jr.CaseName,
			false,
			time.Now().Sub(t0).Seconds(),
			"internal error",
			fmt.Sprintf("runner output invalid: %v", err),
		}
	}
	return result
}

func removeAllVerbose(directory string) {
	log.Println("Removing temporary directory", directory)
	os.RemoveAll(directory)
}

func judge(ctx context.Context, rule Rule, cases []string) []*pb.Result {
	var exe string
	if rule.SkipCompile {
		exe = rule.Target
	} else {
		buildDir := tempdir()
		defer removeAllVerbose(buildDir)
		if !compile(ctx, rule, buildDir) {
			return nil
		}
		exe = filepath.Join(buildDir, rule.Target)
	}

	requests := make(chan judgeRequest)
	responses := make(chan judgeResult)

	for i := 0; i < 4; i++ {
		go func() {
			for r := range requests {
				responses <- judgeCase(ctx, r)
			}
		}()
	}

	go func() {
		for caseID, casename := range cases {
			for i := 0; i < rule.MedianOf; i++ {
				requests <- judgeRequest{
					CaseID:     caseID,
					CaseName:   casename,
					Executable: exe,
					Runner:     rule.Runner,
					Debug:      rule.Debug,
				}
			}
		}
		close(requests)
	}()

	caseWidth := 0
	for _, casename := range cases {
		if caseWidth < len(casename) {
			caseWidth = len(casename)
		}
	}

	printResult := func(result judgeResult, hint string) {
		log.Printf("%*s%s %7.2f   %s",
			caseWidth,
			result.CaseName,
			hint,
			result.Time,
			result.formatDescription(),
		)
	}

	result := make([]*pb.Result, 0, len(cases))
	buffer := make([][]judgeResult, len(cases))
	medNumWidth := len(strconv.Itoa(rule.MedianOf))
	I := len(cases) * rule.MedianOf
	for i := 0; i < I; i++ {
		select {
		case <-ctx.Done():
			return result
		case response := <-responses:
			if rule.MedianOf > 1 {
				buffer[response.CaseID] = append(buffer[response.CaseID], response)
				printResult(response, fmt.Sprintf("#%0*d", medNumWidth, len(buffer[response.CaseID])))
				if len(buffer[response.CaseID]) == rule.MedianOf {
					runs := buffer[response.CaseID]
					sort.Slice(runs, func(i, j int) bool {
						if runs[i].Passed == runs[j].Passed {
							return runs[i].Time < runs[j].Time
						}
						return runs[i].Passed
					})
					result = append(result, runs[rule.MedianOf/2].toProto())
					printResult(runs[rule.MedianOf/2], fmt.Sprintf(" %*s", medNumWidth, ""))
				}
			} else {
				printResult(response, "")
				result = append(result, response.toProto())
			}
		}
	}
	return result
}

// Options is passed to MainOptions
type Options struct {
	Chdir        string
	ExcludeCases []string // exclude these cases
	IncludeCases []string // include these cases
	AsUser       string   // run the judge as this user. privileged.
	RuleFile     string   // use the rules defined in the config file instead of argv[0]. privileged.
	Server       string   // the judge server
	Homework     string   // the name of the homework
	Bin          string   // skip compiling and use the given binary. privileged.
	MedianOf     int      // run each case multiple times and pick the median as the result
	Debug        bool     // output debug messages
}

// MainOptions runs the judge with the given options
func MainOptions(options *Options) {
	if options.Chdir != "" {
		err := os.Chdir(options.Chdir)
		if err != nil {
			log.Fatalf("failed to chdir: %v", err)
		}
	}
	if !strings.HasPrefix(options.Server, "unix://") && strings.ContainsRune(options.Server, '/') {
		options.Server = "unix://" + options.Server
	}
	conn, err := grpc.Dial(options.Server,
		grpc.WithInsecure(), grpc.WithBlock(), grpc.FailOnNonTempDialError(true))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())

	interrupted := make(chan os.Signal, 1)
	go func() {
		<-interrupted
		log.Println("Cleaning up...")
		cancel()
	}()
	signal.Notify(interrupted, os.Interrupt)

	c := pb.NewScoreboardClient(conn)
	var hw *pb.Homework
	if options.RuleFile != "" {
		if sb.Privileged() {
			hw = sb.LoadHomework(options.RuleFile)
		} else {
			log.Fatal("Cannot specify rule file when not privileged")
		}
	} else {
		hw, err = c.QueryHomework(ctx, &pb.QueryHomeworkRequest{
			Name: options.Homework,
		})
		if err != nil {
			log.Fatalf("failed to get homework %s: %v", options.Homework, err)
		}
	}

	if options.AsUser != username && !sb.Privileged() {
		log.Fatal("Cannot run as other user when not privileged")
	}

	cases := make([]string, 0, len(hw.Cases))
	for _, kase := range hw.Cases {
		keep := true
		for _, ex := range options.ExcludeCases {
			if kase == ex {
				keep = false
			}
		}
		if len(options.IncludeCases) > 0 && len(options.ExcludeCases) == 0 {
			keep = false
		}
		for _, in := range options.IncludeCases {
			if kase == in {
				keep = true
			}
		}
		if keep {
			cases = append(cases, kase)
		} else {
			log.Println("Excluded", kase)
		}
	}

	if options.MedianOf%2 == 0 {
		log.Fatal("Refusing to pick a median from a even number of runs")
	}

	rule := Rule{
		Target:   hw.Target,
		Runner:   hw.Runner,
		Optional: make([]OptionalFile, len(hw.Files)),
		MedianOf: options.MedianOf,
		Debug:    options.Debug,
	}

	for i, source := range hw.Files {
		rule.Optional[i].Name = source.Name
		rule.Optional[i].Fallback = source.Fallback
	}

	if options.Bin != "" {
		if sb.Privileged() {
			rule.SkipCompile = true
			rule.Target = options.Bin
		} else {
			log.Println("Cannot skip compiling when not privileged")
		}
	}

	result := judge(ctx, rule, cases)
	if len(result) == 0 {
		return
	}
	cancel()

	sbCtx, sbCancel := context.WithTimeout(context.Background(), time.Second*3)
	r, err := c.Submit(sbCtx, &pb.UserSubmission{
		User:     options.AsUser,
		Homework: hw.Name,
		Results:  result,
	})
	if err != nil {
		log.Fatalf("failed to submit results to scoreboard: %v", err)
	}
	log.Println("Scoreboard:", r.Message)
	sbCancel()
}
