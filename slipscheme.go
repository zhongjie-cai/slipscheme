package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Schema represents JSON schema.
type Schema struct {
	Title             string             `json:"title,omitempty"`
	ID                string             `json:"id,omitempty"`
	Type              SchemaType         `json:"type,omitempty"`
	Description       string             `json:"description,omitempty"`
	Definitions       map[string]*Schema `json:"definitions,omitempty"`
	Properties        map[string]*Schema `json:"properties,omitempty"`
	PatternProperties map[string]*Schema `json:"patternProperties,omitempty"`
	Ref               string             `json:"$ref,omitempty"`
	Items             *Schema            `json:"items,omitempty"`
}

func (schema *Schema) String() string {
	var bytes, err = json.Marshal(schema)
	if err != nil {
		return ""
	}
	return string(bytes)
}

var anonymousObjectCount = 0

// Name will attempt to determine the name of the Schema element using
// the Title or ID (in that order)
func (schema *Schema) Name() string {
	name := schema.Title
	if name == "" {
		name = schema.ID
	}
	return name
}

// SchemaType is an ENUM that is set on parsing the schema
type SchemaType int

const (
	// ANY is a schema element that has no defined type
	ANY SchemaType = iota
	// ARRAY is a schema type "array"
	ARRAY SchemaType = iota
	// BOOLEAN is a schema type "boolean"
	BOOLEAN SchemaType = iota
	// INTEGER is a schema type "integer"
	INTEGER SchemaType = iota
	// NUMBER is a schema type "number"
	NUMBER SchemaType = iota
	// NULL is a schema type "null"
	NULL SchemaType = iota
	// OBJECT is a schema type "object"
	OBJECT SchemaType = iota
	// STRING is a schema type "string"
	STRING SchemaType = iota
)

// UnmarshalJSON for SchemaType so we can parse the schema
// type string and set the ENUM
func (s *SchemaType) UnmarshalJSON(b []byte) error {
	var schemaType string
	err := json.Unmarshal(b, &schemaType)
	if err != nil {
		return err
	}
	types := map[string]SchemaType{
		"array":   ARRAY,
		"boolean": BOOLEAN,
		"integer": INTEGER,
		"number":  NUMBER,
		"null":    NULL,
		"object":  OBJECT,
		"string":  STRING,
	}
	if val, ok := types[schemaType]; ok {
		*s = val
		return nil
	}
	return fmt.Errorf("Unknown schema type \"%s\"", schemaType)
}

// MarshalJSON for SchemaType so we serialized the schema back
// to json for debugging
func (s *SchemaType) MarshalJSON() ([]byte, error) {
	switch *s {
	case ANY:
		return []byte("\"object\""), nil
	case ARRAY:
		return []byte("\"array\""), nil
	case BOOLEAN:
		return []byte("\"boolean\""), nil
	case INTEGER:
		return []byte("\"integer\""), nil
	case NUMBER:
		return []byte("\"number\""), nil
	case NULL:
		return []byte("\"null\""), nil
	case OBJECT:
		return []byte("\"object\""), nil
	case STRING:
		return []byte("\"string\""), nil
	}
	return nil, fmt.Errorf("Unknown Schema Type: %#v", s)
}

func getFileList(args []string) []string {
	var fileList []string
	for _, arg := range args {
		var matches, err = filepath.Glob(arg)
		if err != nil {
			return fileList
		}
		fileList = append(fileList, matches...)
	}
	return fileList
}

func getReferenceName(file string) string {
	_, name := path.Split(file)
	return name
}

func main() {
	outputDir := flag.String("dir", "tmp", "output directory for go files.")
	pkgName := flag.String("pkg", "model", "package namespace for go files")
	overwrite := flag.Bool("overwrite", true, "force overwriting existing go files")
	stdout := flag.Bool("stdout", false, "print go code to stdout rather than files")
	format := flag.Bool("fmt", true, "pass code through gofmt")
	comments := flag.Bool("comments", true, "enable/disable print comments")

	flag.Parse()

	if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
		os.MkdirAll(*outputDir, 0755)
	}

	processor := &SchemaProcessor{
		OutputDir:   *outputDir,
		PackageName: *pkgName,
		Overwrite:   *overwrite,
		Stdout:      *stdout,
		Fmt:         *format,
		Comment:     *comments,
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s <schema file> [<schema file> ...]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	files := getFileList(args)
	err := processor.Load(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	err = processor.Process()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// SchemaProcessor object used to convert json schemas to golang structs
type SchemaProcessor struct {
	OutputDir   string
	PackageName string
	Overwrite   bool
	Stdout      bool
	Fmt         bool
	Comment     bool
	schemas     map[string]*Schema
	processed   map[string]bool
}

// Load will read a list of json schema files and concert to schema objects
func (s *SchemaProcessor) Load(files []string) error {
	s.schemas = make(map[string]*Schema)
	for _, file := range files {
		var fh *os.File
		var err error
		fh, err = os.OpenFile(file, os.O_RDONLY, 0644)
		defer fh.Close()
		if err != nil {
			return err
		}
		b, err := ioutil.ReadAll(fh)
		if err != nil {
			return err
		}

		reference := getReferenceName(file)
		schema, err := s.LoadSchema(b, reference)
		if err != nil {
			return err
		}

		s.schemas[reference] = schema
	}
	return nil
}

// Process will read a list of json schema files, parse them
// and write them to the OutputDir
func (s *SchemaProcessor) Process() error {
	var targetSchemas []*Schema
	for key, schema := range s.schemas {
		targetSchema, err := s.ParseSchema(key, schema)
		if err != nil {
			return err
		}
		targetSchemas = append(targetSchemas, targetSchema)
	}
	for _, targetSchema := range targetSchemas {
		_, err := s.processSchema(targetSchema)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateDefinitionTitles(schema *Schema) {
	if schema.Definitions != nil {
		for k, v := range schema.Definitions {
			if v.Name() == "" {
				v.Title = k
			}
			updateDefinitionTitles(v)
		}
	}
}

func updateReferencePath(schema *Schema, reference string) {
	if schema.Definitions != nil {
		for _, v := range schema.Definitions {
			updateReferencePath(v, reference)
		}
	}
	if schema.Properties != nil {
		for _, v := range schema.Properties {
			updateReferencePath(v, reference)
		}
	}
	if schema.PatternProperties != nil {
		for _, v := range schema.PatternProperties {
			updateReferencePath(v, reference)
		}
	}
	if schema.Items != nil {
		updateReferencePath(schema.Items, reference)
	}
	if schema.Ref != "" {
		schemaPath := strings.Split(schema.Ref, "/")
		if len(schemaPath) > 0 && schemaPath[0] == "#" {
			schema.Ref = reference + schema.Ref
		}
	}
}

// LoadSchema simply loads the schema.
func (s *SchemaProcessor) LoadSchema(data []byte, reference string) (*Schema, error) {
	schema := &Schema{}
	err := json.Unmarshal(data, schema)
	if err != nil {
		return nil, err
	}
	updateDefinitionTitles(schema)
	updateReferencePath(schema, reference)
	return schema, nil
}

// ParseSchema post-processes the objects
// so as to resolve/flatten any $ref objects
// found in the document.
func (s *SchemaProcessor) ParseSchema(reference string, schema *Schema) (*Schema, error) {
	err := s.resolveRefs(reference, schema)
	if err != nil {
		return nil, err
	}
	s.setTitle(reference, schema)
	return schema, nil
}

func (s *SchemaProcessor) resolveRefs(reference string, schema *Schema) error {
	if schema.Ref != "" {
		schemaPath := strings.Split(schema.Ref, "/")
		var ctx interface{}
		ctx = schema
		for _, part := range schemaPath {
			if part == "#" {
				return errors.New("Invalid reference point - please make sure references have file names specified - " + reference)
			} else if strings.HasSuffix(part, "#") {
				var referenceName = part[:len(part)-1]
				var referenceSchema, found = s.schemas[referenceName]
				if !found {
					return errors.New("Invalid reference file - please make sure the referenced files are in the processing list - " + reference + " ? " + referenceName)
				}
				ctx = referenceSchema
			} else if part == "definitions" {
				ctx = ctx.(*Schema).Definitions
			} else if part == "properties" {
				ctx = ctx.(*Schema).Properties
			} else if part == "patternProperties" {
				ctx = ctx.(*Schema).PatternProperties
			} else if part == "items" {
				ctx = ctx.(*Schema).Items
			} else {
				if cast, ok := ctx.(map[string]*Schema); ok {
					ctx = cast[part]
				}
			}
		}
		if cast, ok := ctx.(*Schema); ok {
			*schema = *cast
		}
		err := s.resolveRefs(reference, schema)
		if err != nil {
			return err
		}
	}

	if schema.Definitions != nil {
		for _, v := range schema.Definitions {
			err := s.resolveRefs(reference, v)
			if err != nil {
				return err
			}
		}
	}
	if schema.Properties != nil {
		for _, v := range schema.Properties {
			err := s.resolveRefs(reference, v)
			if err != nil {
				return err
			}
		}
	}
	if schema.PatternProperties != nil {
		for _, v := range schema.PatternProperties {
			err := s.resolveRefs(reference, v)
			if err != nil {
				return err
			}
		}
	}
	if schema.Items != nil {
		err := s.resolveRefs(reference, schema.Items)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SchemaProcessor) setTitle(reference string, schema *Schema) {
	if schema.Definitions != nil {
		for k, v := range schema.Definitions {
			if v.Name() == "" {
				v.Title = k
			}
			s.setTitle(reference, v)
		}
	}
	if schema.Properties != nil {
		for k, v := range schema.Properties {
			if v.Name() == "" {
				v.Title = k
			}
			s.setTitle(reference, v)
		}
	}
	if schema.PatternProperties != nil {
		for k, v := range schema.PatternProperties {
			if v.Name() == "" {
				v.Title = k
			}
			s.setTitle(reference, v)
		}
	}
	if schema.Items != nil {
		if schema.Items.Name() == "" {
			if schema.Name() == "" {
				anonymousObjectCount++
				schema.Title = fmt.Sprintf("AnonymousObject%v", anonymousObjectCount)
			}
			schema.Items.Title = schema.Name() + "Item"
		}
		s.setTitle(reference, schema.Items)
	}
}

func camelCase(name string) string {
	caseName := strings.Title(
		strings.Map(func(r rune) rune {
			if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				return r
			}
			return ' '
		}, name),
	)
	caseName = strings.Replace(caseName, " ", "", -1)

	for _, suffix := range []string{"Id", "Url", "Json", "Xml"} {
		if strings.HasSuffix(caseName, suffix) {
			return strings.TrimSuffix(caseName, suffix) + strings.ToUpper(suffix)
		}
	}

	for _, prefix := range []string{"Url", "Json", "Xml"} {
		if strings.HasPrefix(caseName, prefix) {
			return strings.ToUpper(prefix) + strings.TrimPrefix(caseName, prefix)
		}
	}

	return caseName
}

func (s *SchemaProcessor) structComment(schema *Schema, typeName string) string {
	if !s.Comment {
		return ""
	}
	prettySchema, _ := json.MarshalIndent(schema, "// ", "  ")
	return fmt.Sprintf("// %s defined from schema:\n// %s\n", typeName, prettySchema)
}

func (s *SchemaProcessor) processSchema(schema *Schema) (typeName string, err error) {
	if schema.Type == OBJECT {
		typeName = camelCase(schema.Name())
		if schema.Properties != nil {
			typeData := fmt.Sprintf("%stype %s struct {\n", s.structComment(schema, typeName), typeName)
			keys := []string{}
			for k := range schema.Properties {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				v := schema.Properties[k]
				subTypeName, err := s.processSchema(v)
				if err != nil {
					return "", err
				}
				typeData += fmt.Sprintf("    %s %s `json:\"%s,omitempty\" yaml:\"%s,omitempty\"`\n", camelCase(k), subTypeName, k, k)
			}
			typeData += "}\n\n"
			if err := s.writeGoCode(typeName, typeData); err != nil {
				return "", err
			}
			typeName = fmt.Sprintf("*%s", typeName)
		} else if schema.PatternProperties != nil {
			keys := []string{}
			for k := range schema.PatternProperties {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				v := schema.PatternProperties[k]
				subTypeName, err := s.processSchema(v)
				if err != nil {
					return "", err
				}

				// verify subTypeName is not a simple type
				if strings.Title(subTypeName) == subTypeName {
					typeName = strings.TrimPrefix(fmt.Sprintf("%sMap", subTypeName), "*")
					typeData := fmt.Sprintf("%stype %s map[string]%s\n\n", s.structComment(schema, typeName), typeName, subTypeName)
					if err := s.writeGoCode(typeName, typeData); err != nil {
						return "", err
					}
				} else {
					typeName = fmt.Sprintf("map[string]%s", subTypeName)
				}
			}
		} else {
			typeName = "map[string]interface{}"
		}
	} else if schema.Type == ARRAY {
		subTypeName, err := s.processSchema(schema.Items)
		if err != nil {
			return "", err
		}

		typeName = camelCase(schema.Name())
		if typeName == "" {
			if strings.Title(subTypeName) == subTypeName {
				if strings.HasSuffix(subTypeName, "s") {
					typeName = fmt.Sprintf("%ses", subTypeName)
				} else {
					typeName = fmt.Sprintf("%ss", subTypeName)
				}
			}
		}
		if typeName != "" {
			typeName = strings.TrimPrefix(typeName, "*")
			typeData := fmt.Sprintf("%stype %s []%s\n\n", s.structComment(schema, typeName), typeName, subTypeName)
			if err := s.writeGoCode(typeName, typeData); err != nil {
				return "", err
			}
		} else {
			typeName = fmt.Sprintf("[]%s", subTypeName)
		}
	} else if schema.Type == BOOLEAN {
		typeName = "bool"
	} else if schema.Type == INTEGER {
		typeName = "int"
	} else if schema.Type == NUMBER {
		typeName = "float64"
	} else if schema.Type == NULL || schema.Type == ANY {
		typeName = "interface{}"
	} else if schema.Type == STRING {
		typeName = "string"
	}
	return
}

func (s *SchemaProcessor) writeGoCode(typeName, code string) error {
	if seen, ok := s.processed[typeName]; ok && seen {
		return nil
	}
	// mark schemas as processed so we dont print/write it out again
	if s.processed == nil {
		s.processed = map[string]bool{
			typeName: true,
		}
	} else {
		s.processed[typeName] = true
	}

	if s.Stdout {
		if s.Fmt {
			cmd := exec.Command("gofmt", "-s")
			inPipe, _ := cmd.StdinPipe()
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Start()
			inPipe.Write([]byte(code))
			inPipe.Close()
			return cmd.Wait()
		}
		fmt.Print(code)
		return nil
	}
	file := path.Join(s.OutputDir, fmt.Sprintf("%s.go", typeName))
	if !s.Overwrite {
		if _, err := os.Stat(file); err == nil {
			log.Printf("File %s already exists, skipping without -overwrite", file)
			return nil
		}
	}
	fh, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	//defer fh.Close()
	preamble := fmt.Sprintf("package %s\n", s.PackageName)
	preamble += fmt.Sprintf(`
/////////////////////////////////////////////////////////////////////////
// This Code is Generated by SlipScheme Project:
// https://github.com/zhongjie-cai/slipscheme
// 
// Generated with command:
// %s
/////////////////////////////////////////////////////////////////////////
//                            DO NOT EDIT                              //
/////////////////////////////////////////////////////////////////////////

`, strings.Join(os.Args, " "))

	if _, err := fh.Write([]byte(preamble)); err != nil {
		return err
	}
	if _, err := fh.Write([]byte(code)); err != nil {
		return err
	}

	if s.Fmt {
		cmd := exec.Command("gofmt", "-s", "-w", file)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return nil
}
