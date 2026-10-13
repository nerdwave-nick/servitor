// Command convert converts the scripture of pattern Mark I in a Librarium
// to pattern Mark II, as every ritual that may write does once Mark I is
// retired, and tells each conversion in one line. The VM trial converts its
// rite with it, so that it judges what the conversion makes, whether or not
// the servitor under trial converts on its own reading.
//
//	go run ./test/vm/convert <Librarium>
//
// Exit: 0 every conversion was written (or none was due), 1 one was
// lamented, 2 no Librarium was named.
package main

import (
	"fmt"
	"os"

	"github.com/nerdwave-nick/servitor/internal/librarium"
	"github.com/nerdwave-nick/servitor/internal/rituals"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: convert <Librarium>")
		os.Exit(2)
	}
	told := rituals.OpenConverting(os.Args[1], librarium.Runes{}).Convert()
	for _, t := range told.Tidings {
		fmt.Println(t)
	}
	for _, l := range told.Laments {
		fmt.Fprintln(os.Stderr, l)
	}
	if len(told.Laments) > 0 {
		os.Exit(1)
	}
}
