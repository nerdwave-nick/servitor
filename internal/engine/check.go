package engine

import (
	"errors"
	"fmt"
	"os"

	"github.com/nerdwave-nick/servitor/internal/config"
	"github.com/nerdwave-nick/servitor/internal/lexicon"
)

// CheckTargets inspects the managed blocks in the target files of sw.
func CheckTargets(l *lexicon.Lexicon, sw *config.Switch) config.Diagnostics {
	var ds config.Diagnostics
	st := ReadStatus(sw)
	for i, fs := range st.Files {
		f := &sw.Files[i]
		d := config.Diagnostic{File: fs.File, Switch: sw.Name, Severity: config.SevWarning}
		switch {
		case fs.Error != "":
			d.Severity, d.Message = config.SevError, fmt.Sprintf("%s %q: %s", l.Guard, fs.Guard, fs.Error)
		case fs.Present && fs.Drift:
			d.Message = l.P(fmt.Sprintf("the sanctum %q was tainted by unsanctioned hands (it differs from the scripture of aspect %q)", fs.Guard, fs.Meta["state"]),
				fmt.Sprintf("managed block %q does not match the configured value for state %q (edited by hand?)", fs.Guard, fs.Meta["state"]))
		case fs.Present:
			continue
		default:
			if _, err := os.Stat(fs.File); errors.Is(err, os.ErrNotExist) {
				if f.Create {
					continue
				}
				d.Message = l.P(fmt.Sprintf("the vessel does not exist (rite %q cannot be performed unless \"create\" is true)", sw.Name),
					fmt.Sprintf("target file does not exist (switch %q cannot be applied unless \"create\" is true)", sw.Name))
			} else {
				continue // block is appended on first apply
			}
		}
		ds = append(ds, d)
	}
	if st.Err != nil && errors.Is(st.Err, ErrInconsistent) && len(ds) == 0 {
		ds = append(ds, config.Diagnostic{File: sw.Path, Switch: sw.Name, Severity: config.SevWarning,
			Message: l.P("the vessels disagree: ", "target files are inconsistent: ") + st.Err.Error()})
	}
	return ds
}
