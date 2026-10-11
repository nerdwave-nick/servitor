package dominion

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestVessel_LiesAmongTheDominionsOfTheStateHomeUnderTheSealOfItsPath(t *testing.T) {
	env := func(k string) (string, bool) {
		if k == "XDG_STATE_HOME" {
			return "/state", true
		}
		return "", false
	}
	sum := sha256.Sum256([]byte("/home/faithful/.config/niri/util.kdl"))
	want := "/state/servitor/dominions/vessels/" + hex.EncodeToString(sum[:])
	if got := Vessel(env, "/home/faithful/.config/niri/util.kdl"); got != want {
		t.Fatalf("the dominion over the vessel lies at %s, not %s", got, want)
	}
	if Vessel(env, "/a") == Vessel(env, "/b") {
		t.Fatal("two vessels share one place of dominion")
	}
}

// TestResolve_FollowsTheTethersOfEveryHallAboveThePlace: the place itself is
// never followed (a sanctum has already followed its vessel's tethers, and a
// tether rebinds the name itself); every hall above it is, and halls that do
// not stand yet are reckoned through the halls above them.
func TestResolve_FollowsTheTethersOfEveryHallAboveThePlace(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(root, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "util.kdl"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for name, anchor := range map[string]string{"linked": real, "relative": "real", "name": "real/util.kdl"} {
		if err := os.Symlink(anchor, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct{ name, path, want string }{
		{"a vessel named directly", filepath.Join(real, "util.kdl"), filepath.Join(real, "util.kdl")},
		{"through a tethered hall", filepath.Join(root, "linked", "util.kdl"), filepath.Join(real, "util.kdl")},
		{"through a hall tethered nearby", filepath.Join(root, "relative", "util.kdl"), filepath.Join(real, "util.kdl")},
		{"an absent vessel in a tethered hall", filepath.Join(root, "linked", "new.kdl"), filepath.Join(real, "new.kdl")},
		{"absent halls beneath a tethered hall", filepath.Join(root, "linked", "a", "b", "new.kdl"),
			filepath.Join(real, "a", "b", "new.kdl")},
		{"the tethered name itself", filepath.Join(root, "name"), filepath.Join(root, "name")},
		{"the tethered name in a tethered hall", filepath.Join(root, "linked", "..", "real", "x"),
			filepath.Join(real, "x")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Resolve(c.path); got != c.want {
				t.Fatalf("Resolve(%s) = %s, not %s", c.path, got, c.want)
			}
		})
	}
}
