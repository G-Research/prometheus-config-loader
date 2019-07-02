package cfgloader

import (
	"path/filepath"
	"testing"
)

func TestBuildRulename(t *testing.T) {
	cases := []struct {
		filename   string
		prometheus string
		expected   string
	}{
		{"blah.yaml", "prom", "prom-blah-rules"},
		{"/tmp/blah.yaml", "prom", "prom-blah-rules"},
		{"/tmp/blah.blah.yaml", "prom", "prom-blah-blah-rules"},
	}

	for ix, td := range cases {
		seen := buildRuleName(td.filename, td.prometheus)
		if td.expected != seen {
			t.Errorf("Case %d, saw %s expected %s", ix, seen, td.expected)
		}
	}
}

func TestLoadDirectory(t *testing.T) {
	cases := []struct {
		directory string
		fail      bool
	}{
		{"testdata", true},
		{"testdata2", false},
		{"angry-wombats", true},
	}

	for ix, test := range cases {
		_, err := LoadConfigurationDirectory(test.directory, "namespace", "prometheus-k8s")
		if (err != nil) != test.fail {
			t.Errorf("Case #%d, loading %s, unexpected error status, err != nil is %v, expected %v", ix, test.directory, (err != nil), test.fail)
			if err != nil {
				t.Errorf("  Actual error is %s", err)
			}
		}
	}
}

func TestLoadConfigurationFile(t *testing.T) {
	cases := []struct {
		filename     string
		namespace    string
		prometheus   string
		expectedName string
		expectedFail bool
	}{
		{"rule1.yaml", "blah", "testprom", "testprom-rule1-rules", false},
		{"rule2.yaml", "bleh", "prom", "prom-rule2-rules", false},
		{"rule3.yaml", "bloop", "foo", "foo-rule3-rules", false},
		{"rule4.yaml", "fail", "fail", "fail-rule4-rules", true},
		{"rule5.yaml", "fail", "fail", "fail-rule5-rules", true},
	}

	for ix, td := range cases {
		seen, err := LoadConfigurationFile(filepath.Join("testdata", td.filename), td.namespace, td.prometheus)

		if (err != nil) != td.expectedFail {
			t.Errorf("Case #%d, unexpected error status, (err != nil) is %v, expected %v", ix, err != nil, td.expectedFail)
			if !td.expectedFail {
				t.Errorf("Case #%d, seen error was %s", ix, err)
			}
		}
		if err == nil {
			if seen.GetName() != td.expectedName {
				t.Errorf("case #%d, saw name %s, expected %s", ix, seen.GetName(), td.expectedName)
			}
			labels := seen.GetLabels()
			if seenProm, ok := labels["prometheus"]; !ok {
				t.Errorf("Case #%d, no prometheus label present", ix)
			} else {
				if seenProm != td.prometheus {
					t.Errorf("case #%d, saw prometheus label %s, expected %s", ix, seenProm, td.prometheus)
				}
			}
		}
	}
}
