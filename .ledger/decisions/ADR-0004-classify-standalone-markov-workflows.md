# ADR-0004: Classify Standalone Markov Workflow YAML

## Status

Accepted

## Context

Project synchronization discovers definitions for the Projects page. Directory
workflows are detected by their `meta.yaml` root and emitted as one directory
entry; files beneath that root are excluded from standalone discovery.

For the remaining repository content, `ListWorkflowDefinitions` currently emits
every file with a `.yaml` or `.yml` extension. A typical source repository also
contains Kubernetes manifests, CI configuration, Ansible content, application
configuration, and other YAML. Presenting all of those files as possible Markov
workflows makes the import list noisy and moves the first semantic check to the
later import operation.

Markov single-file workflows have a recognizable top-level structure. Current
workflow examples contain an `entrypoint` and a `workflows` sequence. Each
workflow has a name and a steps sequence. Optional top-level sections such as
`vars`, `rules`, and `step_types` provide supporting evidence but are not
present in every valid workflow and therefore cannot be required.

Discovery needs to be selective without becoming a second implementation of
Markov validation. Markov remains the authority on whether an imported
definition is executable.

## Decision

Classify standalone YAML files in the project discovery layer before returning
them from `GET /api/v1/projects/{id}/files`.

Directory detection remains the first discovery stage. The standalone
classifier runs only for `.yaml` and `.yml` files that are not inside an already
detected directory workflow.

A file is a standalone Markov workflow candidate when all of the following are
true:

1. The YAML parses successfully as exactly one document.
2. The document root is a mapping.
3. The root has a non-empty scalar `entrypoint`.
4. The root has a non-empty `workflows` sequence.
5. At least one `workflows` item is a mapping with a non-empty scalar `name` and
   a `steps` sequence.
6. An item with that valid workflow shape has a name matching the `entrypoint`
   value.

The classifier is structural and returns a classification result, not a
validation result. Optional `vars`, `rules`, and `step_types` keys do not affect
the positive result. Unknown top-level keys are allowed so discovery remains
forward-compatible with Markov additions.

Files that do not match are omitted from the workflow-definition list. Invalid
YAML and non-matching YAML do not fail the entire project listing operation.
Unexpected filesystem errors still propagate as discovery errors.

Full validation with the configured Markov binary remains mandatory during
import. The classifier must not shell out to Markov for every YAML file during
listing because that would make project browsing scale with process startup and
validation cost.

## Rationale

Requiring only a filename extension has high recall but no useful semantic
precision. Requiring the complete set of `entrypoint`, `vars`, `rules`,
`step_types`, and `workflows` would reject valid workflows whose optional
sections are absent.

The chosen signature uses the two defining single-file concepts and verifies
their relationship. Checking that the entrypoint names a workflow is more
selective than checking for key presence alone, while remaining substantially
cheaper and less coupled than reproducing all Markov validation rules.

Parsing into `yaml.Node` is preferred over unmarshalling into a permissive Go
struct because node kinds allow the classifier to distinguish scalars,
mappings, and sequences explicitly and reject duplicate or multi-document
ambiguity deliberately.

## Consequences

Positive:

- The Projects page lists likely Markov workflows instead of every YAML file.
- Kubernetes, CI, and general configuration YAML are excluded before users can
  select them.
- Discovery remains local, deterministic, and independent of the Markov binary.
- Import-time Markov validation continues to protect the workflow catalog.

Negative:

- A future valid Markov single-file shape without an explicit `entrypoint`, or
  with workflows sourced indirectly, will be hidden until the classifier is
  updated.
- A non-Markov document can still pass if it deliberately uses the same
  structure; classification is not validation.
- Project discovery now reads and parses standalone YAML contents instead of
  considering filenames alone.

## Implementation Notes

- Add a small classifier in `internal/projects`, separate from filesystem
  walking and from `workflowdef.ValidateWithMarkov`.
- Return a typed result or boolean plus a stable reason suitable for unit tests
  and debug logging; do not expose parser internals as user-facing API errors.
- Detect duplicate top-level `entrypoint` or `workflows` keys and classify the
  file as ambiguous/non-matching.
- Keep discovery ordering deterministic so filtering does not reorder the
  Projects page unexpectedly.
- Test `.yaml` and `.yml`, malformed and multi-document YAML, non-mapping roots,
  common Kubernetes/CI shapes, missing or mistyped required keys, entrypoint
  mismatch, minimal valid workflows, optional sections, and directory-workflow
  exclusion.
- Preserve import-time validation and legacy file import requests. A client may
  still request a known file path directly, but it must pass Markov validation.

## Related

- [Standalone Markov workflow classification plan](../plans/004-standalone-workflow-classification.md)
- [Workflow input formats decision](ADR-0002-workflow-definition-input-formats.md)
- [Workflow input formats plan](../plans/001-workflow-input-formats.md)
