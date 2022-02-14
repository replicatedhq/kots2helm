[![build-test](https://github.com/replicatedhq/kots2helm/actions/workflows/build-test.yaml/badge.svg)](https://github.com/replicatedhq/kots2helm/actions/workflows/build-test.yaml)

# KOTS2Helm

This is an experimental repo that attempts to convert a KOTS application to a Helm chart.

## What does it do?

Given a directory of Kubernetes and KOTS manifests, this project will parse this and create a working Helm chart.

### Config -> Values

Helm charts require a values.yaml file. KOTS applications used a config.yaml syntax. This utility will convert a KOTS config to a helm values.yaml, keeping the KOTS heriarchy and defaults.

### Template functions

KOTS application use {{repl }} template functions. This utility will convert (some of) these to Helm templates.

The following list contains the tested template functions and how they are converted:

| KOTS Template | Supported | Notes 
|---------------|-----------|------
| ConfigOption | Yes | 
| ConfigOptionEquals | Yes 
| IsKurl | Yes | Always will evaluate to false, this will write a value to values.yaml `isKurl = false` and replace the template function {{ IsKurl }} with {{ .Values.isKurl }}
| Namespace | Yes | Uses the {{ .Release.Namespace }} function

### Annotations

KOTS supports `kots.io/when` and `kots.io/exclude` annotations. These will be converted to {{ if }}... {{ end if}} around the entire manifest.

In addition to the template functions, this will config conditional logic (if, else, end) from {{repl if}} to helm's {{if }} syntax.

### TODO 

- Support for multi doc yaml?
- LicenseFieldValue?

## Example?

Ok, so here's an example:

```
% git clone https://github.com/replicatedhq/kots-sentry
Cloning into 'kots-sentry'...
remote: Enumerating objects: 567, done.
remote: Counting objects: 100% (225/225), done.
remote: Compressing objects: 100% (152/152), done.
remote: Total 567 (delta 117), reused 126 (delta 56), pack-reused 342
Receiving objects: 100% (567/567), 199.03 KiB | 2.52 MiB/s, done.
Resolving deltas: 100% (342/342), done.

% .kots2helm ../kots-sentry/manifests --name sentry --version 0.0.1
Helm chart created at sentry-0.0.1.tgz
```
