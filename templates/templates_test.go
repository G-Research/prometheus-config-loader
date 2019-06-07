package templates

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func compareValues(expected, seen Values) bool {
	if len(expected.Values) != len(seen.Values) {
		return false
	}

	for key, expectedVal := range expected.Values {
		seenVal, ok := seen.Values[key]
		if !ok {
			return false
		}
		if seenVal != expectedVal {
			return false
		}
	}
	for key, seenVal := range seen.Values {
		expectedVal, ok := expected.Values[key]
		if !ok {
			return false
		}
		if seenVal != expectedVal {
			return false
		}
	}
	return true
}

func TestParseValues(t *testing.T) {
	td := []struct {
		yaml     []byte
		expected Values
	}{
		{
			[]byte(""),
			Values{Values: map[string]string{}},
		},
		{
			[]byte("foo: hello"),
			Values{Values: map[string]string{"foo": "hello"}},
		},
		{
			[]byte("foo: 7.3"),
			Values{Values: map[string]string{"foo": "7.3"}},
		},
	}

	for ix, test := range td {
		_, seen, err := parseValues("./test.vars", test.yaml)
		if err != nil {
			t.Errorf("Test case %d, error: %s", ix, err)
		}
		if !compareValues(test.expected, seen) {
			t.Errorf("Test case %d, seen and expected do not match (seen: %v  expected: %v)", ix, seen, test.expected)
		}
	}
}

func TestComposeValues(t *testing.T) {
	a := Values{Values: map[string]string{
		"a": "foo",
		"b": "bar",
		"c": "xyzzy",
	}}
	b := Values{Values: map[string]string{
		"b": "overridden",
	}}
	c := Values{Values: map[string]string{
		"c": "overridden",
	}}

	cases := []struct {
		first    Values
		second   Values
		expected Values
	}{
		{a, b, Values{Values: map[string]string{"a": "foo", "b": "overridden", "c": "xyzzy"}}},
		{a, c, Values{Values: map[string]string{"a": "foo", "b": "bar", "c": "overridden"}}},
		{b, c, Values{Values: map[string]string{"b": "overridden", "c": "overridden"}}},
	}

	for ix, test := range cases {
		seen := mergeValues(test.first, test.second)
		if !compareValues(test.expected, seen) {
			t.Errorf("Test case %d, seen and expected do not match (seen: %v  expected: %v)", ix, seen, test.expected)
		}
	}
}

func compareFiles(seenName, expectedName string) bool {
	seenFile, err := os.Open(seenName)
	if err != nil {
		return false
	}
	expectedFile, err := os.Open(expectedName)
	if err != nil {
		return false
	}

	seen, err := ioutil.ReadAll(seenFile)
	if err != nil {
		return false
	}
	expected, err := ioutil.ReadAll(expectedFile)
	if err != nil {
		return false
	}
	for pos, seenVal := range seen {
		if pos >= len(expected) {
			return false
		}
		if seenVal != expected[pos] {
			return false
		}
	}
	return true
}

func TestInternalTemplateExpansionContext1(t *testing.T) {
	cleanup := true
	DefaultTempDirectory = "testdata/output"
	tpl, err := createInternalTemplate("testdata/testdir1")
	if err != nil {
		t.Errorf("Unexpected error, %s", err)
	}

	if tpl.sourceDir != "testdata/testdir1" {
		t.Errorf("Unexpected source directory, saw %s, expected testdata/testdir1", tpl.sourceDir)
	}

	for _, key := range []string{"default", "context1"} {
		if _, ok := tpl.variables[key]; !ok {
			t.Errorf("context variables for %s not present.", key)
		}
	}

	if len(tpl.variables) != 2 {
		t.Errorf("Unexpected number of context variable settings, expected 2, saw %d", len(tpl.variables))
	}

	context1Data, err := expandDirectory("context1", tpl)
	if err != nil {
		t.Fatalf("Unexpected error creating template expansion directory, %s (data is %v)", err, context1Data)
	}

	for _, name := range context1Data.Files {
		seenName := filepath.Join(context1Data.Directory, name)
		expectedName := filepath.Join("testdata/expected/testdir1-context1", name)
		if !compareFiles(seenName, expectedName) {
			t.Errorf("Unexpected file difference, seen path: %s, expected path = %s, please manually diff", seenName, expectedName)
			cleanup = false
		}
	}

	if cleanup {
		context1Data.Cleanup()
	}
}

func TestInternalTemplateExpansionContextWithRandomDirectory(t *testing.T) {
	cleanup := true
	DefaultTempDirectory = fmt.Sprintf("testdata/output-%d", rand.Uint32())
	tpl, err := createInternalTemplate("testdata/testdir1")
	if err != nil {
		t.Errorf("Unexpected error, %s", err)
	}

	if tpl.sourceDir != "testdata/testdir1" {
		t.Errorf("Unexpected source directory, saw %s, expected testdata/testdir1", tpl.sourceDir)
	}

	for _, key := range []string{"default", "context1"} {
		if _, ok := tpl.variables[key]; !ok {
			t.Errorf("context variables for %s not present.", key)
		}
	}

	if len(tpl.variables) != 2 {
		t.Errorf("Unexpected number of context variable settings, expected 2, saw %d", len(tpl.variables))
	}

	context1Data, err := expandDirectory("context1", tpl)
	if err != nil {
		t.Fatalf("Unexpected error creating template expansion directory, %s (data is %v)", err, context1Data)
	}

	for _, name := range context1Data.Files {
		seenName := filepath.Join(context1Data.Directory, name)
		expectedName := filepath.Join("testdata/expected/testdir1-context1", name)
		if !compareFiles(seenName, expectedName) {
			t.Errorf("Unexpected file difference, seen path: %s, expected path = %s, please manually diff", seenName, expectedName)
			cleanup = false
		}
	}

	if cleanup {
		context1Data.Cleanup()
	}
}
