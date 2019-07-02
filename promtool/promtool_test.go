package promtool

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

// Test the result of missing or invalid promtool
func TestCheckInvalidPromtool(t *testing.T) {
	p := Promtool{
		Executable: "/not/a/valid/promtool",
	}
	_, err := p.Check("")
	expected := PromtoolError{
		Executable: "/not/a/valid/promtool",
		Operation:  "check",
		FileName:   "",
		Stdout:     "",
		Stderr:     "",
		OriginalError: &os.PathError{
			Op:   "fork/exec",
			Path: "/not/a/valid/promtool",
			Err:  errors.New("no such file or directory"),
		},
	}

	perr, ok := err.(PromtoolError)
	if !ok {
		t.Errorf("Expected Check to return PromtoolError, got %s\n", reflect.TypeOf(err).String())
		return
	}

	if !errorsAreSame(expected, perr) {
		t.Errorf("Expected: %s\nGot: %s\n", expected.Error(), perr.Error())
	}

	// Fall through test with reflection
	perr.OriginalError = nil
	expected.OriginalError = nil
	if !reflect.DeepEqual(expected, perr) {
		t.Errorf("CheckInvalidPromtool output error is not in expected format.\n")
	}

}

// This test fails if promtool is not in the PATH
func TestNew(t *testing.T) {
	_, err := New()
	if err != nil {
		t.Errorf("promtool.New() fails: %s\n", err.Error())
	}
}

// Test Promtool Check
func TestCheck(t *testing.T) {
	p, _ := New()

	tests := []struct {
		Filename string
		Valid    bool
	}{
		{Filename: "testdata/valid/valid1.yaml", Valid: true},
		{Filename: "testdata/invalid/invalid1.yaml", Valid: false},
	}

	for _, test := range tests {
		stdout, err := p.Check(test.Filename)
		if test.Valid {
			if err != nil {
				perr := err.(PromtoolError)
				t.Errorf("%s should be valid, but received an error running promtool:\n%s", test.Filename, perr.Stderr)
			}
		} else {
			if err == nil {
				t.Errorf("%s should be invalid, but promtool returned sucess:\n%s", test.Filename, stdout)
			}
		}
	}
}

// Test Promtool Check on a valid directory
func TestCheckDirectory(t *testing.T) {
	p, _ := New()

	tests := []struct {
		Directory string
		Valid     bool
	}{
		{Directory: "testdata/valid", Valid: true},
		{Directory: "testdata/valid/", Valid: true},
		{Directory: "testdata/invalid", Valid: false},
		{Directory: "testdata/invalid/", Valid: false},
		{Directory: "testdata/mixed", Valid: true},
		{Directory: "testdata/mixed/", Valid: true},
	}

	for _, test := range tests {
		err := p.CheckDirectory(test.Directory)
		if test.Valid {
			if err != nil {
				t.Errorf("%s should be a valid directory, but promtool.CheckDirectory failed: %s\n", test.Directory, err.Error())
			}
		} else {
			if err == nil {
				t.Errorf("%s should be an invalid directory, but promtool.CheckDirectoy succeeded.\n", test.Directory)
			}
		}
	}
}

// Test Promtool test on a series of single test files
func TestTest(t *testing.T) {
	p, _ := New()
	wd, _ := os.Getwd()

	tests := []struct {
		Filename string
		Workdir  string
		Valid    bool
	}{
		{Filename: "tests/valid1.yaml", Workdir: wd + "/testdata/valid", Valid: true},
		{Filename: "tests/invalid1.yaml", Workdir: wd + "/testdata/mixed", Valid: false},
		{Filename: "tests/nonexistanttest.yaml", Workdir: wd + "/testdata/invalid", Valid: false},
	}

	for _, test := range tests {
		stdout, err := p.Test(test.Filename, test.Workdir)
		if test.Valid {
			if err != nil {
				t.Errorf("%s should be a passing test, but promtool.Test failed: %s\n", test.Filename, err.Error())
			}
		} else {
			if err == nil {
				t.Errorf("%s should be a failing test, but promtool.Test succeeded.\n%s", test.Filename, stdout)
			}
		}
	}
}

// Test Promtool test on directories
func TestTestDirectory(t *testing.T) {
	p, _ := New()
	wd, _ := os.Getwd()

	tests := []struct {
		Directory string
		Workdir   string
		Valid     bool
	}{
		{Directory: "tests", Workdir: wd + "/testdata/valid", Valid: true},
		{Directory: "tests", Workdir: wd + "/testdata/mixed", Valid: false},
		{Directory: "tests", Workdir: wd + "/testdata/invalid", Valid: false},
	}

	for _, test := range tests {
		err := p.TestDirectory(test.Directory, test.Workdir)
		if test.Valid {
			if err != nil {

				t.Errorf("%s should be a passing test, but promtool.TestDirectory failed: %s\n", test.Workdir, err.Error())
			}
		} else {
			if err == nil {
				t.Errorf("%s should be a failing test, but promtool.TestDirectory succeeded.\n", test.Workdir)
			}
		}
	}
}

// Helper, can't iterate structs without reflection
func errorsAreSame(a, b PromtoolError) bool {
	return a.Error() == b.Error()
}
