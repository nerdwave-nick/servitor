package librarium

import (
	"strings"
	"testing"
)

const riteSchemaURL = "https://raw.githubusercontent.com/nerdwave-nick/servitor/main/schema/rite.schema.json"

// TestParse_SchemaIsAcceptedAndKept: "$schema" guides the faithful's editors;
// the servitor accepts it, heeds it not, and keeps it for writing.
func TestParse_SchemaIsAcceptedAndKept(t *testing.T) {
	r, fs := parse(t, `{"$schema": "`+riteSchemaURL+`", "pattern": "Mark I", "aspects": ["on"]}`)
	if len(fs) != 0 {
		t.Fatalf("findings:\n%v", fs)
	}
	if r.Schema != riteSchemaURL {
		t.Fatalf("schema %q, want %q", r.Schema, riteSchemaURL)
	}
	bare, _ := parse(t, `{"pattern": "Mark I", "aspects": ["on"]}`)
	r.Schema = ""
	if !r.Equal(bare) {
		t.Fatalf("a rite bearing \"$schema\" must differ from its bare twin in nothing else")
	}
}

func TestParse_SchemaMustBeAString(t *testing.T) {
	scripture := `{"pattern": "Mark I", "$schema": 7, "aspects": ["on"]}`
	_, fs := parse(t, scripture)
	got := findingsWith(fs, `the "$schema" of a rite must be written as a string`)
	if len(got) != 1 || got[0].Severity != Heresy || got[0].Position != posOf(t, scripture, `7,`, 0) {
		t.Fatalf("findings:\n%v", fs)
	}
}

func TestMarshal_KeepsTheSchemaFirst(t *testing.T) {
	r := &Rite{Name: "x", Schema: riteSchemaURL, Aspects: []string{"on"}}
	data := string(roundTrip(t, r))
	if want := "{\n  \"$schema\": \"" + riteSchemaURL + "\",\n  \"pattern\": \"Mark I\",\n"; !strings.HasPrefix(data, want) {
		t.Fatalf("scripture must begin with its \"$schema\":\n%s", data)
	}
	if data := string(roundTrip(t, &Rite{Name: "x", Aspects: []string{"on"}})); strings.Contains(data, "$schema") {
		t.Fatalf("a rite without \"$schema\" must be written without it:\n%s", data)
	}
}

// TestParseSettings_SchemaIsAcceptedAndIgnored: the settings bear "$schema"
// as rites do, and it changes no order.
func TestParseSettings_SchemaIsAcceptedAndIgnored(t *testing.T) {
	scripture := `{"$schema": "https://raw.githubusercontent.com/nerdwave-nick/servitor/main/schema/settings.schema.json",
	  "pattern": "Mark I", "vox": "off"}`
	s, fs := ParseSettings(settingsPath, []byte(scripture))
	if len(fs) != 0 {
		t.Fatalf("findings:\n%v", fs)
	}
	if *s != (Settings{Path: settingsPath, Vox: VoxOff}) {
		t.Fatalf("settings %+v", *s)
	}
	_, fs = ParseSettings(settingsPath, []byte(`{"pattern": "Mark I", "$schema": null}`))
	if len(findingsWith(fs, `the "$schema" of the settings must be written as a string`)) != 1 {
		t.Fatalf("findings:\n%v", fs)
	}
}
