package templates

import (
	"fmt"
	yaml "gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// TemplateData contains various information about template expansions
type TemplateData struct {
	// Name of the kubernetes context this set of template expansions is for
	Context string
	// Name of the base directory for this expansion
	Directory string
	// A list of all files that exist in the directory, uni tests
	// are prefixed with the string "tests/"
	Files []string
}

// Values is a data structure that encapsulates the variables from a
// settings file.
type Values struct {
	Values map[string]string
}

// internalTemplate packages up the templates and variables for a directory.
type internalTemplate struct {
	variables map[string]Values
	templates map[string]*template.Template
	sourceDir string
}

// ExpansionData is simply a map from "context name" to a TemplateData
// structure containing the relevant information for the expanded
// templates for that context.
type ExpansionData map[string]TemplateData

// Default directory for template expansion output directories.
var DefaultTempDirectory = "/tmp/prometheus-config-loader"

// ExpandDirectory takes a list of context names and a base directory,
// then calls expandDirectory for each context, collating all the data
// and returning it.
func ExpandDirectory(contexts []string, sourceDirectory string) (ExpansionData, error) {
	rv := make(ExpansionData)

	templates, err := createInternalTemplate(sourceDirectory)
	if err != nil {
		return rv, err
	}
	for _, context := range contexts {
		data, err := expandDirectory(context, templates)
		if err != nil {
			return nil, err
		}
		rv[context] = data
	}

	return rv, nil
}

// createInternalTemplate parses all templates in a directory, reads
// all the variables settings and returns a structure encapsulating
// these in a form suitable for later consumption.
func createInternalTemplate(directory string) (internalTemplate, error) {
	rv := internalTemplate{sourceDir: directory}
	rv.templates = make(map[string]*template.Template)
	rv.variables = make(map[string]Values)

	names, err := filepath.Glob(filepath.Join(directory, "*.vars"))
	if err != nil {
		return rv, err
	}
	for _, name := range names {
		context, values, err := readValues(name)
		if err != nil {
			return rv, err
		}
		rv.variables[context] = values
	}

	names, err = filepath.Glob(filepath.Join(directory, "*.yaml"))
	if err != nil {
		return rv, err
	}
	for _, name := range names {
		base := filepath.Base(name)
		tmpl := template.New(base).Delims("<{[", "]}>")
		tmpl, err := tmpl.ParseFiles(name)
		if err != nil {
			return rv, err
		}
		rv.templates[base] = tmpl
	}

	return rv, nil
}

// getVariables returns the context-specific Values struct for a
// specific context (or an empty Values, of no named context exists).
func (i internalTemplate) getVariables(context string) Values {
	rv, ok := i.variables[context]
	if ok {
		return rv
	}
	return Values{}
}

// readValues reads a context variables file, returning the context
// name, a values containing the parsed variables and an error if one
// occurs.
func readValues(name string) (string, Values, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", Values{}, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return "", Values{}, err
	}

	return parseValues(name, data)
}

// parseValues expects YAML data in a []byte and returns the parsed
// version. If an error is returned, nil and the error from
// yaml.Unmarshal is returned.
func parseValues(name string, data []byte) (string, Values, error) {
	name = filepath.Base(name)
	extStart := strings.Index(name, ".vars")
	if extStart >= 0 {
		name = name[:extStart]
	}
	rv := Values{Values: make(map[string]string)}
	err := yaml.Unmarshal(data, &rv.Values)
	if err != nil {
		return name, Values{}, err
	}

	return name, rv, nil
}

// mergeValues merges two Values dictionaries, letting anything set in
// the second override anything set in the first.
func mergeValues(first, second Values) Values {
	rv := Values{Values: make(map[string]string)}

	for key, val := range first.Values {
		rv.Values[key] = val
	}
	for key, val := range second.Values {
		rv.Values[key] = val
	}

	return rv
}

// Create a temporary directory, ensuring that our "base" exists,
// cretaing it if needed.
func mktemp(base, leaf string) (string, error) {
	_, err := os.Stat(base)
	if os.IsNotExist(err) {
		err = os.MkdirAll(base, 0755)
		if err != nil {
			return "", err
		}
	}

	return ioutil.TempDir(base, leaf)
}

// expandDirectory takes a context name, and an internalTemplate
// structure, then proceeds to create an output directory, read the
// context-specific configuration (first by reading the file
// default.vars (if it exists) and then <context>.vars (overriding any
// variables set from the default) and template-expanding the files
// matching *.yaml into the output directory.
//
// Once that is complete, it will simply copy files matching
// tests/*.yaml into the tests/ subdirectory of the output directory.
func expandDirectory(context string, data internalTemplate) (TemplateData, error) {
	values := mergeValues(data.getVariables("default"), data.getVariables(context))
	values.Values["context"] = context
	rv := TemplateData{}

	outDir, err := mktemp(DefaultTempDirectory, fmt.Sprintf("tmp-%s-", context))
	if err != nil {
		return rv, err
	}
	rv.Directory = outDir
	rv.Context = context

	for filename, tpl := range data.templates {
		out, err := os.Create(filepath.Join(outDir, filename))
		if err != nil {
			return rv, err
		}
		defer out.Close()
		rv.Files = append(rv.Files, filename)
		err = tpl.Execute(out, values)
		if err != nil {
			return rv, err
		}
	}

	os.Mkdir(filepath.Join(outDir, "tests"), 0777)
	testFiles, err := filepath.Glob(filepath.Join(data.sourceDir, "tests", "*.yaml"))
	if err != nil {
		return rv, err
	}
	for _, file := range testFiles {
		source, err := os.Open(file)
		defer source.Close()
		if err != nil {
			return rv, err
		}
		sink, err := os.Create(filepath.Join(outDir, "tests", filepath.Base(file)))
		defer sink.Close()
		if err != nil {
			return rv, err
		}
		io.Copy(sink, source)
		rv.Files = append(rv.Files, filepath.Join("tests", filepath.Base(file)))
	}

	return rv, nil
}

// Clean up the output from a template expansion
func (t TemplateData) Cleanup() {
	os.RemoveAll(t.Directory)
}
