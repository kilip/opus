//go:build ignore

package main

import (
	"encoding/json"
	"os"

	"github.com/invopop/jsonschema"
	"github.com/kilip/opus/server/internal/config"
)

func main() {
	r := new(jsonschema.Reflector)
	schema := r.Reflect(&config.Config{})

	out, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile("../../../docs/config.schema.json", out, 0644); err != nil {
		panic(err)
	}
}
