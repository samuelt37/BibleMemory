package notes

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Segment struct {
	Offset  int `json:"o"` // byte offset into raw_text
	Seconds int `json:"s"`
}

var tsRe = regexp.MustCompile(`^\s*\[(?:(\d+):)?(\d{1,2}):(\d{2})\]\s*`)
var anyTsRe = regexp.MustCompile(`\[(?:(\d+):)?(\d{1,2}):(\d{2})\]\s*`)

func tsSeconds(m []string) int {
	h, _ := strconv.Atoi(m[1]) // empty -> 0
	mm, _ := strconv.Atoi(m[2])
	ss, _ := strconv.Atoi(m[3])
	return h*3600 + mm*60 + ss
}

func formatTS(total int) string {
	h, m, s := total/3600, total%3600/60, total%60
	if h > 0 {
		return fmt.Sprintf("[%d:%02d:%02d]", h, m, s)
	}
	return fmt.Sprintf("[%02d:%02d]", m, s)
}

func idxSeconds(s string, m []int) int {
	num := func(a, b int) int {
		if a < 0 {
			return 0
		}
		n, _ := strconv.Atoi(s[a:b])
		return n
	}
	return num(m[2], m[3])*3600 + num(m[4], m[5])*60 + num(m[6], m[7])
}

func offsetTimestamps(text string, offsetSec int) string {
	if offsetSec == 0 {
		return text
	}
	return anyTsRe.ReplaceAllStringFunc(text, func(s string) string {
		return formatTS(tsSeconds(anyTsRe.FindStringSubmatch(s))+offsetSec) + " "
	})
}

func parseTimestamped(text string) (string, []Segment) {
	var b strings.Builder
	var segs []Segment
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			continue
		}
		last := 0
		for _, m := range anyTsRe.FindAllStringSubmatchIndex(line, -1) {
			b.WriteString(line[last:m[0]])
			segs = append(segs, Segment{Offset: b.Len(), Seconds: idxSeconds(line, m)})
			last = m[1]
		}
		rest := strings.TrimSpace(line[last:])
		b.WriteString(rest)
		if strings.HasPrefix(rest, "[Slide:") {
			b.WriteString("\n")
		} else {
			b.WriteString(" ")
		}
	}
	return strings.TrimSpace(b.String()), segs
}

// secondsAt returns the time of the last marker at or before offset.
func secondsAt(segs []Segment, offset int) *int {
	if len(segs) == 0 {
		return nil
	}
	i := sort.Search(len(segs), func(i int) bool { return segs[i].Offset > offset })
	s := segs[0].Seconds
	if i > 0 {
		s = segs[i-1].Seconds
	}
	return &s
}

// findFrom locates a chunk in the raw text by its first ~60 bytes.
func findFrom(raw string, from int, content string) int {
	probe := strings.TrimSpace(content)
	if len(probe) > 60 {
		probe = probe[:60]
	}
	if probe == "" || from > len(raw) {
		return -1
	}
	i := strings.Index(raw[from:], probe)
	if i < 0 {
		return -1
	}
	return from + i
}
