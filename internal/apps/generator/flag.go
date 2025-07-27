package generator

import "flag"

type Flags struct {
	// Path to the output file for the generated schema.
	GenerateSchemaFile string
}

func LoadFlag() *Flags {
	cfg := Flags{}
	flag.StringVar(&cfg.GenerateSchemaFile, "output", "labtime-configuration-schema.json", "Path to the output file for the generated schema")
	flag.Parse()
	return &cfg
}
