package sanction

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nerdwave-nick/servitor/internal/librarium"
)

// subject adds the exalted subject v of a state rite: a sanctum,
// transcription or tether with the fields of its verse of Mark I under the
// keys of its designation, a machine-spirit with the omen of the plan and
// the enact words of every omen it declares.
func (s *shower) subject(v Verse) {
	s.add(2, "subject %s · %s · place %d", word(v.Subject), v.Designation(), v.Place+1)
	if !v.Spirit() {
		key := map[librarium.Kind]string{librarium.KindSanctum: "vessel", librarium.KindTranscription: "vessel",
			librarium.KindTether: "tether"}[v.Kind]
		s.body(v, key)
		return
	}
	s.field("omen", v.Omen)
	for _, omen := range sortedKeys(v.Enact) {
		s.field("enact "+word(omen), v.Enact[omen])
	}
	s.speech(v)
}

// observance adds the exalted observance v of a state rite: its speech, its
// guard "from", and the places of the subjects that name it.
func (s *shower) observance(v Verse) {
	s.add(2, "observance %s · %s", word(v.Observance), v.Kind.Key())
	key := map[librarium.Kind]string{librarium.KindIncantation: "words", librarium.KindLitany: "scroll"}[v.Kind]
	s.body(v, key)
	from := "(every aspect)"
	if v.From != nil {
		from = quoted(v.From)
	}
	s.field("from", from)
	s.field("introit of the subjects at", places(v.Introit))
	s.field("recessional of the subjects at", places(v.Recessional))
}

// places are the places of subjects, counted from one.
func places(at []int) string {
	if len(at) == 0 {
		return "(none)"
	}
	out := make([]string, len(at))
	for i, p := range at {
		out[i] = strconv.Itoa(p + 1)
	}
	return strings.Join(out, ", ")
}

func quoted(words []string) string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = strconv.Quote(w)
	}
	return fmt.Sprint(strings.Join(out, ", "))
}
