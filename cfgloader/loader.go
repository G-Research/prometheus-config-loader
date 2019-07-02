// cfgloader is a package providing code to load prometheus rule
// definitions into prometheus-operator PrometheusRule CRDs
package cfgloader

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/G-Research/prometheus-config-loader/cfgloader/rulefmt"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// LoadConfigurationDirectory loads all the YAML files in a directory
// and reurns a PrometheusRuleList object, suitable for sending to a
// kubernetes API server.
//
// It will generate these to be suitable for the named prometheus, in
// the given namespace.
//
// If any errors occur while loading the individual PrometheusRules,
// the prometheusRule will not be added, and the last error that
// occured will be the error returned from the function.
func LoadConfigurationDirectory(directory, namespace, prometheus string) (*v1.PrometheusRuleList, error) {
	var errSeen error = nil
	glob := filepath.Join(directory, "*.yaml")
	names, err := filepath.Glob(glob)
	if err != nil {
		return nil, err
	}
	if names == nil {
		return nil, fmt.Errorf("Unable to expand directory %s", directory)
	}

	rv := v1.PrometheusRuleList{}

	for _, name := range names {
		rule, err := LoadConfigurationFile(name, namespace, prometheus)
		if err != nil {
			errSeen = err
		} else {
			rv.Items = append(rv.Items, rule)
		}
	}

	return &rv, errSeen
}

// Construct the PrometheusRule name based on the prometheus it is
// for, and the base file name of the rules. This expects that the
// file name ends in ".yaml".
func buildRuleName(fileName, prometheus string) string {
	base := filepath.Base(fileName)
	base = strings.ReplaceAll(base[:len(base)-5], ".", "-")
	return fmt.Sprintf("%s-%s-rules", prometheus, base)
}

// LoadConfigurationFile loads a configuration file into a
// PrometheusRule object, with the name and namespace set, as well as
// setting the prometheus label to the name of the prometheus it is
// intended for.
//
// In case of an error occuring, the returned PrometheusRule could be
// nil, working, or in a broken state.
func LoadConfigurationFile(name, namespace, prometheus string) (*v1.PrometheusRule, error) {
	f, err := os.Open(name)
	defer f.Close()
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	spec, err := ParseRuleSpec(data)
	if err != nil {
		return nil, err
	}

	rv := &v1.PrometheusRule{Spec: spec}
	rv.SetNamespace(namespace)
	rv.SetName(buildRuleName(name, prometheus))
	labels := map[string]string{"prometheus": prometheus, "role": "prometheus-rulefiles"}
	rv.SetLabels(labels)

	return rv, err
}

func ParseRuleSpec(data []byte) (v1.PrometheusRuleSpec, error) {
	var intermediate rulefmt.RuleGroups
	var rv v1.PrometheusRuleSpec

	err := yaml.Unmarshal(data, &intermediate)

	if err != nil {
		return rv, err
	}

	if len(intermediate.Groups) == 0 {
		// TODO: consider creating a custom error?
		return rv, errors.New("No groups found")
	}

	for _, g := range intermediate.Groups {
		rg := v1.RuleGroup{Name: g.Name, Interval: g.Interval}
		for _, r := range g.Rules {
			tmp := v1.Rule{
				Record:      r.Record,
				Alert:       r.Alert,
				Expr:        intstr.FromString(r.Expr),
				For:         r.For,
				Labels:      r.Labels,
				Annotations: r.Annotations,
			}
			rg.Rules = append(rg.Rules, tmp)
		}
		rv.Groups = append(rv.Groups, rg)
	}

	return rv, nil
}
