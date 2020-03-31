package intrange

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Range expands the given integer range specified in a string into a slice of string integers
func Range(s string) (expanded []string, err error) {
	parts := strings.Split(s, ",")
	for _, part := range parts {
		splitted := strings.SplitN(part, "-", 2)
		if len(splitted) == 1 {
			expanded = append(expanded, part)
		} else {
			var i, j uint64
			i, err = strconv.ParseUint(splitted[0], 0, 64)
			if err != nil {
				return nil, fmt.Errorf("bad range: %s", part)
			}
			j, err = strconv.ParseUint(splitted[1], 0, 64)
			if err != nil {
				return nil, fmt.Errorf("bad range: %s", part)
			}
			width := len(splitted[0])
			for ; i <= j; i++ {
				expanded = append(expanded, fmt.Sprintf("%0*d", width, i))
			}
		}
	}
	return
}

// MustRange is like Range, but panics instead of returning an error
func MustRange(s string) []string {
	vals, err := Range(s)
	if err != nil {
		panic(err)
	}
	return vals
}

// Expand expands the string
func Expand(s string) ([]string, error) {
	var parts []string
	var rangestrs []string
	for {
		i := strings.IndexRune(s, '[')
		if i == -1 {
			break
		}
		parts = append(parts, s[:i])
		s = s[i+1:]
		i = strings.IndexRune(s, ']')
		if i == -1 {
			return nil, errors.New("opening `[` without encolsing `]`")
		}
		rangestrs = append(rangestrs, s[:i])
		s = s[i+1:]
	}
	if len(parts) == 0 {
		return []string{s}, nil
	}
	ranges := make([][]string, len(rangestrs))
	for i, rangestr := range rangestrs {
		vals, err := Range(rangestr)
		if err != nil {
			return nil, err
		}
		ranges[i] = vals
	}
	rcounts := make([]int, len(rangestrs))
	var result []string
	for rcounts[0] < len(ranges[0]) {
		var builder strings.Builder
		for i := range parts {
			builder.WriteString(parts[i])
			builder.WriteString(ranges[i][rcounts[i]])
		}
		builder.WriteString(s)
		result = append(result, builder.String())
		rcounts[len(rcounts)-1]++
		for i := len(rcounts) - 1; i > 0; i-- {
			if rcounts[i] == len(ranges[i]) {
				rcounts[i] = 0
				rcounts[i-1]++
			} else {
				break
			}
		}
	}
	return result, nil
}

// MustExpand is like Expand, but panics instead of returning an error
func MustExpand(s string) []string {
	vals, err := Expand(s)
	if err != nil {
		panic(err)
	}
	return vals
}
