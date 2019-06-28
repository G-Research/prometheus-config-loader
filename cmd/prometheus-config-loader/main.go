// A tool to take a directory of Prometheus rule YAML files, and 0 or
// more context-specific settings, perform template expansion on them,
// verify them with promtool, then (given that everything so far has
// been successful) send them to the the specified Kubernetes
// clusters.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/client/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"

	"github.com/G-Research/prometheus-config-loader/cfgloader"
	"github.com/G-Research/prometheus-config-loader/promtool"
	"github.com/G-Research/prometheus-config-loader/templates"
)

const unitTestContextName = "unittest"

// Load a Kubernetes configuration;. We're using the heavier-weight
// configuration, since we need to cross-check context names for later
// use.
func loadKubeConfig(kubeconfig string) *clientapi.Config {
	rv, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		log.Fatalf("Failed to parse kubernetes configuration, %s", err)
	}

	return rv
}

func doSyntaxChecks(prom *promtool.Promtool, tplData templates.ExpansionData) error {
	for _, tpl := range tplData {
		log.Printf("Syntax-checking context %s (dir %s)", tpl.Context, tpl.Directory)
		for _, file := range tpl.Files {
			if !strings.HasPrefix(file, "tests/") {
				out, err := prom.Check(filepath.Join(tpl.Directory, file))
				if err != nil {
					log.Println("Syntax checking failed,")
					fmt.Println(out)
					return err
				}
			}
		}
	}
	return nil
}

func doUnitTests(prom *promtool.Promtool, tplData templates.ExpansionData) error {
	for _, tpl := range tplData {
		if tpl.Context != unitTestContextName {
			continue
		}
		log.Printf("Unit-testing context %s (dir %s)", tpl.Context, tpl.Directory)
		for _, file := range tpl.Files {
			if strings.HasPrefix(file, "tests/") {
				out, err := prom.Test(filepath.Join(tpl.Directory, file), tpl.Directory)
				if err != nil {
					log.Println("Syntax checking failed,")
					fmt.Println(out)
					return err
				}
			}
		}
	}
	return nil
}

// Ensure all specified contexts are valid for the configuration
func validateContexts(cfg *clientapi.Config, contexts []string) bool {
	for _, ctx := range contexts {
		_, ok := cfg.Contexts[ctx]
		if !ok {
			return false
		}
	}

	return true
}

func uploadPrometheusRules(t templates.TemplateData, c *clientapi.Config, dryRun bool, prometheus string, namespace string) {
	if t.Context == unitTestContextName {
		return
	}

	rules, err := cfgloader.LoadConfigurationDirectory(t.Directory, namespace, prometheus)
	if err != nil {
		log.Printf("ERROR: Failed to load prometheus rules from directory %s: %s", t.Directory, err)
		return
	}
	if dryRun {
		log.Printf("INFO: dry-run enabled, emitting loaded rules for context %s", t.Context)
		buf, err := json.MarshalIndent(rules, "", "  ")
		if err != nil {
			log.Printf("ERROR: marshalling to JSON failed: %s", err)
		} else {
			fmt.Println(string(buf))
		}
		return
	}

	overrides := clientcmd.ConfigOverrides{
		Context:        *(c.Contexts[t.Context]),
		CurrentContext: t.Context,
	}

	cc, err := clientcmd.NewDefaultClientConfig(*c, &overrides).ClientConfig()
	if err != nil {
		log.Printf("ERROR: failed to create API client configuration for context %s: %s", t.Context, err)
		return
	}

	api, err := monitoringv1.NewForConfig(cc)
	if err != nil {
		log.Printf("ERROR: failed to connect to API server for context %s: %s", t.Context, err)
		return
	}

	for _, rule := range rules.Items {
		log.Printf("Uploading rule %s to namespace %s, in context %s", rule.GetName(), namespace, t.Context)
		_, err := api.MonitoringV1().PrometheusRules(namespace).Create(rule)
		if err != nil {
			p, err := api.MonitoringV1().PrometheusRules(namespace).Get(rule.GetName(), metav1.GetOptions{})
			if err != nil {
				log.Fatalf("Failed to get %s when trying to update: %s", rule.GetName(), err)
			}
			rule.SetResourceVersion(p.GetResourceVersion())
			_, err = api.MonitoringV1().PrometheusRules(namespace).Update(rule)
			if err != nil {
				oldBuf, _ := json.MarshalIndent(p, "", "  ")
				newBuf, _ := json.MarshalIndent(rule, "", "  ")
				fmt.Printf("Existing:\n%s\n\nNew:\n%s\n", string(oldBuf), string(newBuf))
				log.Fatalf("Failed to create or update, %s", err)
			}
		}
	}
}

// Get the kubernetes config file based on environment variables,
// passed-in flag values, and other defaulting methods.
//
// This returns the passed-in string, if it exists, then falls back to the KUBECONFIG environment variable, then finally a terminal callback.
func kubeConfigFile(flagval string) string {
	if flagval != "" {
		// Always trust explicitly-defined command line flags
		return flagval
	}

	filename, ok := os.LookupEnv("KUBECONFIG")
	if ok {
		// next, if we have the KUBECONFIG environment
		// variable, use that as our result.
		return filename
	}
	return filepath.Join(homedir.HomeDir(), ".kube", "config")
}

func main() {
	// TODO: It might be good to have a more intelligent "find the
	// config file" function, but this is probably good enough for
	// now.
	kubeflag := flag.String("kubeconfig", "", "Kubernetes configuration file, set this to the empty string if the command is running inside a kubernetes pod.")
	flagContexts := flag.String("contexts", "", "Comma-separated list of contexts to push configuration to.")
	prometheus := flag.String("prometheus", "", "Name of the prometheus to push configuration for.")
	namespace := flag.String("namespace", "", "The namespace we should create PrometheusRule objects in.")
	dryRun := flag.Bool("dry-run", false, "Skip uploading, instead print the resulting PrometheusRuleList to stdout.")
	skipSyntax := flag.Bool("skip-syntax-check", false, "Bypass syntax checks of the source prometheus configuration.")
	skipUnits := flag.Bool("skip-unit-tests", false, "Bypass running prometheus unit tests.")

	flag.Parse()

	kubeconfig := kubeConfigFile(*kubeflag)
	contexts := strings.Split(*flagContexts, ",")
	cfg := loadKubeConfig(kubeconfig)
	if !validateContexts(cfg, contexts) {
		log.Print("Failed to validate passed-in contexts.")
		log.Print("Contexts specified that do not exist in the configuration:")
		for _, ctx := range contexts {
			_, ok := cfg.Contexts[ctx]
			if !ok {
				log.Printf("    %s", ctx)
			}
		}
		os.Exit(1)
	}

	// All configured and basic validation done. Next, template expansion.
	if len(flag.Args()) == 0 {
		log.Fatalf("No source directory specified.")
	}
	sourceDir := flag.Arg(0)
	contexts = append([]string{unitTestContextName}, contexts...)
	log.Printf("About to template-expand %s", sourceDir)
	tplData, err := templates.ExpandDirectory(contexts, sourceDir)
	if err != nil {
		log.Fatalf("Failed to expand directories, %s", err)
	}

	// We should now have all our templates expanded.
	prom, err := promtool.New()
	if err != nil {
		log.Fatalf("Failed to find promtool, %s", err)
	}

	// Start with syntax checks
	if *skipSyntax {
		log.Printf("WARNING: syntax-checking is disabled.")
	} else {
		err = doSyntaxChecks(prom, tplData)
		if err != nil {
			log.Fatalf("Failed syntax-checking:\n%s", err)
		}
	}

	// Then, unit-tests
	if *skipUnits {
		log.Printf("WARNING: unit-testing is disabled.")
	} else {
		err = doUnitTests(prom, tplData)
		if err != nil {
			log.Fatalf("Failed unit-testing:\n%s", err)
		}
	}

	// We should now be good to go
	for _, ctx := range contexts {
		tpl := tplData[ctx]
		uploadPrometheusRules(tpl, cfg, *dryRun, *prometheus, *namespace)
	}
}
