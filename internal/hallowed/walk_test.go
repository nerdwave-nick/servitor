package hallowed

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestRead_ReadsAScrollBeneathHallowedHalls(t *testing.T) {
	tp := newTemple(t)
	tp.hall("/etc/conf.d", 0o755)
	tp.scroll("/etc/conf.d/split-lock", "kernel.split_lock_mitigate=0\n", 0o640)
	got, err := tp.g.Read("/etc/conf.d/split-lock")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "kernel.split_lock_mitigate=0\n" {
		t.Fatalf("read %q", got)
	}
}

// TestActs_RefuseHallsOthersMayWrite: every hall along the path, the
// ground's own included, is written by none but its owner.
func TestActs_RefuseHallsOthersMayWrite(t *testing.T) {
	for _, mode := range []os.FileMode{0o775, 0o757, 0o777 | os.ModeSticky} {
		t.Run(fmt.Sprintf("%o", mode), func(t *testing.T) {
			tp := newTemple(t)
			tp.hall("/etc/open", 0o755)
			tp.scroll("/etc/open/vessel", "old", 0o644)
			if err := os.Chmod(tp.at("/etc/open"), mode); err != nil {
				t.Fatal(err)
			}
			if _, err := tp.g.Read("/etc/open/vessel"); !errors.Is(err, errOpenHall) {
				t.Fatalf("read beneath an open hall: %v", err)
			}
			if _, err := tp.g.Transcribe("/etc/open/vessel", []byte("new"), nil); !errors.Is(err, errOpenHall) {
				t.Fatalf("transcribed beneath an open hall: %v", err)
			}
			if _, err := tp.g.Transcribe("/etc/open/new", []byte("new"), nil); !errors.Is(err, errOpenHall) {
				t.Fatalf("brought a vessel into being beneath an open hall: %v", err)
			}
			if got := tp.content("/etc/open/vessel"); got != "old" || !tp.absent("/etc/open/new") {
				t.Fatalf("the open hall was written: %q", got)
			}
		})
	}
	t.Run("the ground itself", func(t *testing.T) {
		tp := newTemple(t)
		tp.scroll("/vessel", "old", 0o644)
		if err := os.Chmod(tp.root, 0o770); err != nil {
			t.Fatal(err)
		}
		if _, err := tp.g.Read("/vessel"); !errors.Is(err, errOpenHall) {
			t.Fatalf("read beneath an open ground: %v", err)
		}
	})
}

// TestActs_RefuseHallsTheRootDoesNotOwn: whom the ground holds to be the
// root decides; here it is a stranger, not the owner of the halls.
func TestActs_RefuseHallsTheRootDoesNotOwn(t *testing.T) {
	tp := newTemple(t)
	tp.hall("/etc", 0o755)
	tp.scroll("/etc/vessel", "old", 0o644)
	tp.g.Sovereign = func(uid uint32) bool { return uid == tp.me.UID+1 }
	if _, err := tp.g.Read("/etc/vessel"); !errors.Is(err, errUnsovereign) {
		t.Fatalf("read beneath halls of another: %v", err)
	}
	if _, err := tp.g.Transcribe("/etc/vessel", []byte("new"), nil); !errors.Is(err, errUnsovereign) {
		t.Fatalf("transcribed beneath halls of another: %v", err)
	}
	if tp.content("/etc/vessel") != "old" {
		t.Fatal("the vessel was written")
	}
}

// TestActs_RefuseABondIntoAnOpenHall: a bond is walked to where it leads,
// and the halls there are judged like any other.
func TestActs_RefuseABondIntoAnOpenHall(t *testing.T) {
	tp := newTemple(t)
	tp.hall("/etc", 0o755)
	tp.hall("/home/faithful", 0o757)
	tp.scroll("/home/faithful/vessel", "old", 0o644)
	tp.bond("/etc/through", "/home/faithful")
	tp.bond("/etc/vessel", "../home/faithful/vessel")
	for _, path := range []string{"/etc/through/vessel", "/etc/vessel"} {
		if _, err := tp.g.Transcribe(path, []byte("new"), nil); !errors.Is(err, errOpenHall) {
			t.Fatalf("transcribed %s: %v", path, err)
		}
		if _, err := tp.g.Read(path); !errors.Is(err, errOpenHall) {
			t.Fatalf("read %s: %v", path, err)
		}
	}
	if tp.content("/home/faithful/vessel") != "old" {
		t.Fatal("the vessel in the open hall was written")
	}
}

// TestWalk_FollowsBondsWithinTheGround: bonds are read and walked anew, at
// most forty; absolute bonds and ".." never lead beyond the ground.
func TestWalk_FollowsBondsWithinTheGround(t *testing.T) {
	tp := newTemple(t)
	tp.hall("/usr/lib", 0o755)
	tp.hall("/etc", 0o755)
	tp.scroll("/usr/lib/os-release", "NAME=Forge World\n", 0o644)
	tp.bond("/etc/os-release", "../usr/lib/os-release")
	tp.bond("/etc/absolute", "/usr/lib/os-release")
	tp.bond("/etc/lib", "/usr/lib")
	tp.bond("/etc/climbing", "../../../../usr/lib/os-release")
	tp.bond("/etc/0", "/usr/lib/os-release")
	for i := 1; i <= 40; i++ {
		tp.bond(fmt.Sprintf("/etc/%d", i), fmt.Sprint(i-1))
	}
	tp.bond("/etc/loop", "loop")
	for _, path := range []string{"/etc/os-release", "/etc/absolute", "/etc/lib/os-release",
		"/etc/lib/../lib/./os-release", "/etc/climbing", "/../../etc/os-release", "//etc//os-release", "/etc/39"} {
		got, err := tp.g.Read(path)
		if err != nil || string(got) != "NAME=Forge World\n" {
			t.Errorf("read %s: %q, %v", path, got, err)
		}
	}
	p, err := tp.g.walk("/etc/lib/os-release", vessel)
	if err != nil {
		t.Fatal(err)
	}
	defer p.close()
	if p.path() != "/usr/lib/os-release" {
		t.Errorf("the walk reached %s", p.path())
	}
	for _, path := range []string{"/etc/40", "/etc/loop"} {
		if _, err := tp.g.Read(path); !errors.Is(err, errBondLoop) {
			t.Errorf("read %s: %v", path, err)
		}
	}
}

func TestActs_RefusePathsThatDoNotBeginAtTheRoot(t *testing.T) {
	tp := newTemple(t)
	if _, err := tp.g.Read("etc/vessel"); !errors.Is(err, errAstray) {
		t.Fatalf("read a relative path: %v", err)
	}
	if _, err := tp.g.Transcribe("etc/vessel", nil, nil); !errors.Is(err, errAstray) {
		t.Fatalf("transcribed a relative path: %v", err)
	}
	if _, err := tp.g.Bind("etc/vessel", "/x", false); !errors.Is(err, errAstray) {
		t.Fatalf("bound a relative name: %v", err)
	}
}

// TestActs_RefuseStrangersAtTheVessel: only scrolls are read or
// transcribed; a lone pipe is never opened for a reader that would block.
func TestActs_RefuseStrangersAtTheVessel(t *testing.T) {
	tp := newTemple(t)
	tp.hall("/etc/hall", 0o755)
	if err := unix.Mkfifo(tp.at("/etc/pipe"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/etc/pipe", "/etc/hall", "/etc/hall/..", "/"} {
		if _, err := tp.g.Read(path); !errors.Is(err, errNoScroll) {
			t.Errorf("read %s: %v", path, err)
		}
		if _, err := tp.g.Transcribe(path, []byte("new"), nil); !errors.Is(err, errNoScroll) {
			t.Errorf("transcribed %s: %v", path, err)
		}
	}
	if _, err := tp.g.Read("/etc/absent"); !errors.Is(err, errMissing) {
		t.Errorf("read an absent vessel: %v", err)
	}
	if _, err := tp.g.Read("/etc/absent/vessel"); !errors.Is(err, errMissing) {
		t.Errorf("read beneath an absent hall: %v", err)
	}
}

func TestRead_RefusesMoreScriptureThanItsCapacity(t *testing.T) {
	tp := newTemple(t)
	tp.scroll("/full", "", 0o644)
	if err := os.Truncate(tp.at("/full"), Capacity); err != nil {
		t.Fatal(err)
	}
	if got, err := tp.g.Read("/full"); err != nil || len(got) != Capacity {
		t.Fatalf("read a vessel at capacity: %d bytes, %v", len(got), err)
	}
	if err := os.Truncate(tp.at("/full"), Capacity+1); err != nil {
		t.Fatal(err)
	}
	if _, err := tp.g.Read("/full"); !errors.Is(err, errOverflow) {
		t.Fatalf("read beyond capacity: %v", err)
	}
}

// TestMachine_WalksTheRealHalls: the walk upon the machine itself, read
// only. /etc is the root's; the temporary halls lie beneath /tmp, which all
// may write.
func TestMachine_WalksTheRealHalls(t *testing.T) {
	got, err := Machine.Read("/etc/passwd")
	if err != nil || !strings.Contains(string(got), "root:") {
		t.Fatalf("read /etc/passwd: %d bytes, %v", len(got), err)
	}
	p, err := Machine.walk("/etc/../etc/./passwd", vessel)
	if err != nil {
		t.Fatal(err)
	}
	if p.path() != "/etc/passwd" || p.stat == nil {
		t.Errorf("the walk reached %s", p.path())
	}
	p.close()
	if _, err := os.Lstat("/etc/os-release"); err == nil {
		if _, err := Machine.Read("/etc/os-release"); err != nil {
			t.Errorf("read /etc/os-release: %v", err)
		}
	}
	if _, err := Machine.Read("/dev/null"); !errors.Is(err, errNoScroll) {
		t.Errorf("read /dev/null: %v", err)
	}
	tp := newTemple(t)
	tp.scroll("/vessel", "old", 0o644)
	if _, err := Machine.Read(tp.at("/vessel")); !errors.Is(err, errOpenHall) && !errors.Is(err, errUnsovereign) {
		t.Errorf("read a vessel in the temporary halls: %v", err)
	}
}
