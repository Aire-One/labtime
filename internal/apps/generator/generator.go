package generator

import (
	"encoding/json"
	"os"

	"aireone.xyz/labtime/internal/yamlconfig"
	"github.com/invopop/jsonschema"
)

func GenerateSchema(schema *string) error {
	s := jsonschema.Reflect(&yamlconfig.YamlConfig{})
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	*schema = string(data)
	return nil
}

func WriteToFile(schema string, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(schema); err != nil {
		return err
	}

	return nil
}
