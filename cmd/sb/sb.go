package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NTHU-lsalab/sb"
	"github.com/NTHU-lsalab/sb/pb"

	"github.com/spf13/pflag"
	"google.golang.org/grpc"
)

// Score is used to rank BoardEntries
type Score struct {
	NumPassed   int
	TotalTime   float64
	PenaltyTime float64
}

func (s Score) String() string {
	return fmt.Sprintf("{%d %.2f}", s.NumPassed, s.TotalTime)
}

// Better returns whether the score s is better than o
func (s Score) Better(o Score) bool {
	if s.NumPassed == o.NumPassed {
		return s.TotalTime < o.TotalTime
	}
	return s.NumPassed > o.NumPassed
}

func caseMapFromHomework(hw *pb.Homework) map[string]int {
	caseMap := make(map[string]int)
	for i, casename := range hw.Cases {
		caseMap[casename] = i
	}
	return caseMap
}

func calcScore(hw *pb.Homework, results []*pb.Result) (s Score) {
	caseMap := caseMapFromHomework(hw)
	stats := make([]struct {
		Passed bool
		Time   float64
	}, len(hw.Cases))
	for _, result := range results {
		i, ok := caseMap[result.Case]
		if !ok {
			continue
		}
		if result.Passed {
			stats[i].Time = result.Time
			stats[i].Passed = true
		}
	}
	for _, stat := range stats {
		if stat.Passed {
			s.NumPassed++
			s.TotalTime += stat.Time
		} else {
			s.PenaltyTime += hw.PenaltyTime
		}
	}
	return
}

// BoardEntry = Score + Submission
type BoardEntry struct {
	Score
	Submission *pb.StoredSubmission
}

// Board contains all the information for a homework
type Board struct {
	Homework       *pb.Homework
	submissions    map[string]BoardEntry
	submissionLock sync.Mutex
}

func isStudent(username string) bool {
	return strings.HasPrefix(username, "ipc20")
}

// Rows is for use in template
func (b *Board) Rows() []TableRow {
	rows := make([]TableRow, 0, len(b.submissions))
	for _, boardEntry := range b.submissions {
		rows = append(rows, TableRow{
			BoardEntry: boardEntry,
			Cells:      make([]TableCell, len(b.Homework.Cases)),
		})
	}
	caseMap := caseMapFromHomework(b.Homework)
	for rowi, row := range rows {
		for _, result := range row.BoardEntry.Submission.Results {
			if casei, ok := caseMap[result.Case]; ok {
				rows[rowi].Cells[casei].result = result
			}
		}
	}
	sort.Slice(
		rows,
		func(i, j int) bool { return rows[i].Score.Better(rows[j].Score) },
	)
	rank := 0
	for i := range rows {
		if isStudent(rows[i].Submission.User) {
			rank++
			rows[i].rank = rank
		} else {
			rows[i].rank = -1
		}
	}

	for i := range b.Homework.Cases {
		best := math.Inf(1)
		for _, row := range rows {
			if !isStudent(row.Submission.User) {
				continue
			}
			r := row.Cells[i].result
			if r != nil {
				if r.Passed && r.Time < best {
					best = r.Time
				}
			}
		}
		for _, row := range rows {
			if !isStudent(row.Submission.User) {
				continue
			}
			r := row.Cells[i].result
			if r != nil {
				if r.Passed && r.Time-0.1 < best {
					row.Cells[i].best = true
				}
			}
		}
	}
	return rows
}

// TableRow is a helper object use in html template
type TableRow struct {
	BoardEntry
	rank  int
	Cells []TableCell
}

// Rank returns the rank of the row, or "-" if unapplicable
func (tr TableRow) Rank() string {
	if tr.rank < 0 {
		return "—"
	}
	return strconv.Itoa(tr.rank)
}

// TableCell is a helper object in a html template which corresponds to a <td>
type TableCell struct {
	result *pb.Result
	best   bool
}

// Class returns the class attribute of the <td>
func (tc TableCell) Class() string {
	if tc.best {
		return "best"
	}
	r := tc.result
	if r == nil {
		return "empty"
	}
	if !r.Passed {
		return "failed"
	}
	return ""
}

// Value returns the value enclosed within the <td> tag
func (tc TableCell) Value() string {
	r := tc.result
	if r == nil {
		return "—"
	}
	return fmt.Sprintf("%.2f", r.Time)
}

// Title returns the title attribute of the <td> tag
func (tc TableCell) Title() string {
	if tc.result == nil {
		return "not submitted"
	}
	return tc.result.Verdict
}

func (b *Board) renderBoard() {
	t0 := time.Now()
	b.Rows()
	dirname := filepath.Join(outputDir, b.Homework.Name)
	err := os.MkdirAll(dirname, 0755)
	if err != nil {
		log.Printf("Failed to create directory: %s: %v", dirname, err)
	}
	filename := filepath.Join(dirname, "index.html")
	w, err := os.Create(filename + "-")
	if err != nil {
		log.Printf("Failed to open %s: %v", filename, err)
		return
	}
	defer w.Close()
	err = htmlTemplate.Execute(w, b)
	if err != nil {
		log.Printf("Failed to render %s: %v", filename, err)
		return
	}
	err = os.Rename(filename+"-", filename)
	if err != nil {
		log.Printf("Failed to move: %s: %v", filename, err)
	}
	t1 := time.Now()
	log.Printf("Rendered %s: %d submissions in %s",
		b.Homework.Name, len(b.submissions), t1.Sub(t0))
}

func (b *Board) updateSubmission(new *pb.UserSubmission) string {
	b.submissionLock.Lock()
	defer b.submissionLock.Unlock()
	old, ok := b.submissions[new.User]
	newScore := calcScore(b.Homework, new.Results)
	if !ok || newScore.Better(old.Score) { // new <= old
		b.submissions[new.User] = BoardEntry{
			Score: newScore,
			Submission: &pb.StoredSubmission{
				User:    new.User,
				Results: new.Results,
			},
		}

		storeErr := storeSubmission(b.Homework.Name, b.submissions[new.User].Submission)
		if storeErr != nil {
			log.Printf("Failed to store submission %s/%s: %v", new.Homework, new.User, storeErr)
		}

		b.renderBoard()

		if !ok {
			return fmt.Sprintf("created %v", newScore)
		}
		return fmt.Sprintf("updated %v --> %v", old.Score, newScore)
	}
	return fmt.Sprintf("not updating %v -x-> %v", old.Score, newScore)
}

type server struct {
	boards map[string]*Board
}

var _ pb.ScoreboardServer = &server{}

func loadBoard(hw *pb.Homework) *Board {
	b := &Board{
		Homework:    hw,
		submissions: make(map[string]BoardEntry),
	}
	hwDir := filepath.Join("storage", hw.Name)
	err := os.MkdirAll(hwDir, 0755)
	if err != nil {
		log.Fatalf("Could not create directory for homework %s: %v", hw.Name, err)
	}
	glob, err := filepath.Glob(filepath.Join(hwDir, "*.json"))
	if err != nil {
		panic(err) // malformed glob
	}
	for _, filename := range glob {
		be := BoardEntry{
			Submission: &pb.StoredSubmission{},
		}
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Printf("Failed to read stored submission %s: %v", filename, err)
		}
		err = json.Unmarshal(data, be.Submission)
		if err != nil {
			log.Printf("Failed to load stored submission %s: %v", filename, err)
		}
		be.Score = calcScore(hw, be.Submission.Results)
		b.submissions[be.Submission.User] = be
	}
	b.renderBoard()
	return b
}

func newServer() *server {
	s := &server{
		boards: make(map[string]*Board),
	}
	glob, err := filepath.Glob("config/*.toml")
	if err != nil {
		panic(err) // malformed glob
	}
	for _, filename := range glob {
		hw := sb.LoadHomework(filename)
		s.boards[hw.Name] = loadBoard(hw)
	}
	return s
}

func (s *server) updateSubmission(new *pb.UserSubmission) (string, error) {
	board, ok := s.boards[new.Homework]
	if !ok {
		return "", fmt.Errorf("No such homework: %q", new.Homework)
	}
	return board.updateSubmission(new), nil
}

func (s *server) handleSubmit(ctx context.Context, sub *pb.UserSubmission) (rep *pb.SubmissionReply, err error) {
	msg, err := s.updateSubmission(sub)
	if err != nil {
		return
	}
	rep = &pb.SubmissionReply{Message: msg}
	return
}

func (s *server) Submit(ctx context.Context, sub *pb.UserSubmission) (*pb.SubmissionReply, error) {
	rep, err := s.handleSubmit(ctx, sub)
	if err == nil {
		log.Printf("Accepted %s/%s: %s", sub.Homework, sub.User, rep.Message)
	} else {
		log.Printf("Refused %s/%s: %v", sub.Homework, sub.User, err)
	}
	return rep, err
}

func (s *server) QueryHomework(ctx context.Context, req *pb.QueryHomeworkRequest) (*pb.Homework, error) {
	b, ok := s.boards[req.Name]
	if !ok {
		return nil, errors.New("No such homework")
	}
	return b.Homework, nil
}

var serverAddress string
var outputDir string

func init() {
	pflag.StringVar(&serverAddress, "address", sb.DefaultAddr,
		"the address of the server to listen to. "+
			"If it contains a slash, it is treated as a unix domain socket, "+
			"otherwise it is treated as a tcp socket")
	pflag.StringVar(&outputDir, "outputdir", "out", "html output directory")
}

func main() {
	pflag.Parse()

	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Fatalf("failed to create output directory %s: %v", outputDir, err)
	}
	err = os.MkdirAll(sb.StorageDir, 0755)
	if err != nil {
		log.Fatalf("failed to create storage directory %s: %v", sb.StorageDir, err)
	}
	network := "tcp"
	if strings.ContainsRune(serverAddress, '/') {
		network = "unix"
		err = os.Remove(serverAddress)
		if err != nil && !os.IsNotExist(err) {
			log.Fatalf("failed to remove existing unix socket: %v", err)
		}
	}
	lis, err := net.Listen(network, serverAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	if network == "unix" {
		err = os.Chmod(serverAddress, 0660)
		if err != nil {
			log.Fatalf("failed to set unix socket permission")
		}
	}
	gs := grpc.NewServer()
	s := newServer()
	pb.RegisterScoreboardServer(gs, s)
	if err := gs.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
