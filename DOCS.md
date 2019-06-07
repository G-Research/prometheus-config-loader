# prometheus-config-loader

A tool for loading prometheus rules into prometheus-operator
PrometheusRule CRDs, with (some) guarantees that things will continue
to work.

## The design rationale

This tool was designed to fulfill one very specific need. However, we think
this is a need shared by many

We run multiple clusters, with almost-identical system prometheus
configurations, only varying in very small details like 'cluster name'
and (in a few cases) some alert thresholds that differ depending on
the performance of underlying hardware.

This means we want a templating engine, but we had no wish to
implement one. Thankfully, Go comes with a pretty powerful templating
engine. Unfortunately, prometheus alert descriptions and summaries
uses the same templating engine. But, if we can change the delimiters, we can
avoid really awkward escaping.

We also think rules should be unit-tested, and it should be impossible
to accidentally deploy rules that have failing unit tests or syntax
errors, which is why this tool calls out to promtool to do these checks.

We also think that it's bad if you end up with a PrometheusRule CRD that
is not parseable, so to avoid this, we actually create PrometheusRules
using the prometheus-operator-provided CRD, through the Kubernetes API.

There are cases where these may need to be disabled, however, so there are
checks for that.

## Directory structure

The tool expects a directory full of rule files, wit hhte unit tests
in a subdirectory names "tests". A hypothetical rules directory is
sketched below:

```
config-directory/
  context1.vars
  default.vars
  rule1.yaml
  rule2.yaml
  tests/
    rule1test.yaml
    rule2test.yaml
```

As far as naming goes, it is expected that all rule files and all unit
test files match the glob `*.yaml`.

## Command documentation

General form: `prometheus-config-loader <flags>... <rule directory>`

### Templates

The rule files in the "top-level" directory will be template-expanded.
The templating language is (essentially) Go templates, but using
`<{[` and `]}>` instead of `{{` and `}}`.

Values for expansion come from one of three places. Most are set in
`default.vars`, then overridden by any value set in
`<context>.vars`. The exception is `.Values.context,` the value of that
is set from the kubernetes context for which templates are being
expanded. Unit-testing is done with a (fake) context named `unittest`.

### Flags

| flag | description |
|-----:|:------------|
| --contexts | A comma-separated lists of the context names you want to push rule(s) to. |
| --dry-run | Run throuhg hte normal process, but instead of sending the rules 
| --kubeconfig | Path of youtr kubernetes config file (defaults to $HOME/.kube/config |
| --namespace | Namespace you want the rules created in. |
| --prometheus | Name of the prometheus you are pushing configurations for. |
| --skip-syntax-check | Do not run the syntax-checking |
| --skip-unit-tests | Do not run the unit tests |to a kubernetes API server, simply print them (as JSON) to stdout. |

