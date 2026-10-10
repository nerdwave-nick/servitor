package codex

import (
	"fmt"
	"slices"
	"strings"
)

// IndexText is the index of the codex as the servitor recites it: every
// chapter with each topic and its epigraph.
func IndexText() string {
	var b strings.Builder
	b.WriteString("+++ THE CODEX · every passage the servitor may expound +++\n\n")
	b.WriteString("Every ritual, rune, step and key of the servitor has its passage here, and\n")
	b.WriteString("each passage bears an example. Recite one with: servitor expound <topic>\n")
	width := 0
	for _, name := range Topics() {
		width = max(width, len(name))
	}
	for _, ch := range Index() {
		if len(ch.Passages) == 0 {
			continue
		}
		b.WriteString("\n" + ch.Title + "\n")
		for _, p := range ch.Passages {
			fmt.Fprintf(&b, "  %-*s  %s\n", width, p.Topic, p.Epigraph)
		}
	}
	return b.String()
}

// Suggest names the listed topics that name nearly resembles: those it
// begins, and those within two letters of it.
func Suggest(name string) []string {
	var out []string
	for _, topic := range Topics() {
		if (len(name) >= 3 && strings.HasPrefix(topic, name)) || distance(name, topic) <= 2 {
			out = append(out, topic)
		}
	}
	return slices.Compact(out)
}

// distance is the Levenshtein distance between a and b.
func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
