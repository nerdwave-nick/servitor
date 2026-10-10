// Package schema holds the schemas of pattern Mark I, written in JSON Schema
// (draft 2020-12): one for the scripture of a rite, one for the settings.
// The faithful's editors name them under "$schema" by their raw address in
// the repository; the servitor recites them with expound --schema. Both are
// embedded exactly as written in this directory.
package schema

import (
	"bytes"
	_ "embed"
)

// Names of the schemas, in the order of the codex.
const (
	Rite     = "rite"
	Settings = "settings"
)

// Names lists every schema the servitor bears.
var Names = []string{Rite, Settings}

// Base is the raw address of the repository's schema directory.
const Base = "https://raw.githubusercontent.com/nerdwave-nick/servitor/main/schema/"

var (
	//go:embed rite.schema.json
	rite []byte
	//go:embed settings.schema.json
	settings []byte
)

// File is the name of the schema's file in this directory.
func File(name string) string { return name + ".schema.json" }

// URL is the raw address by which scripture names the schema.
func URL(name string) string { return Base + File(name) }

// For returns the schema named name, exactly as written in the repository.
func For(name string) ([]byte, bool) {
	switch name {
	case Rite:
		return bytes.Clone(rite), true
	case Settings:
		return bytes.Clone(settings), true
	}
	return nil, false
}
