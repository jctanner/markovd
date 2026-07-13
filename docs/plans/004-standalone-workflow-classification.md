# Standalone Markov Workflow Classification Plan

## Goal

Limit project import candidates to detected directory workflows and standalone
YAML files whose structure identifies them as likely Markov workflows.

## Execution Status

Implemented on 2026-07-12. Project discovery now parses remaining standalone
YAML after directory-root exclusion, returns only structurally classified
Markov candidates, and preserves Markov validation during import.

## Motivation

The Projects page currently receives every `.yaml` and `.yml` file outside a
detected directory workflow. Repositories commonly contain many unrelated YAML
files, so the import chooser presents false candidates that will only fail when
the user tries to import them.

Directory workflow discovery is already efficient: a `meta.yaml` root becomes
one definition and its descendants are not listed separately. This plan adds a
structural classification pass for the remaining standalone YAML files.

The governing behavior is defined in
[ADR-0004](../decisions/ADR-0004-classify-standalone-markov-workflows.md).

## Current Behavior

`internal/projects.ListWorkflowDefinitions` performs two broad operations:

1. Walk the repository to find directories containing `meta.yaml` and skip
   their descendants.
2. Call `ListYAMLFiles`, then emit every remaining `.yaml` or `.yml` path as a
   file workflow definition.

`internal/api.handleListProjectFiles` adds imported state and returns those
entries to `ui/src/pages/Projects.tsx`. The UI renders the server response
without inspecting file contents. Full validation occurs only after selection,
through `workflowdef.ValidateWithMarkov` in the import handler.

## Desired Behavior

The listing endpoint should return:

- detected directory workflow roots
- standalone files that match the Markov single-file structural signature

It should omit:

- YAML files inside a detected directory workflow
- Kubernetes manifests and Helm values
- CI, deployment, policy, and application configuration YAML
- malformed YAML
- partial Markov fragments such as an isolated workflow item with `name` and
  `steps` but no top-level `entrypoint` and `workflows`

The UI contract does not need to change. Returned candidates keep the existing
`path`, `kind`, `name`, and `imported` fields.

## Classification Contract

Parse each eligible file as a YAML node tree and classify it as a candidate
only when:

- there is exactly one YAML document
- the document root is a mapping
- `entrypoint` is a non-empty scalar
- `workflows` is a non-empty sequence
- at least one workflow item has a non-empty scalar `name` and a `steps`
  sequence
- a workflow with that name-and-steps shape matches the entrypoint

Optional sections such as `vars`, `rules`, and `step_types` are neither required
nor sufficient. Unknown keys are allowed.

Classification failures are expected filtering outcomes. Filesystem failures
remain errors. Import continues to invoke Markov validation, including when an
API client imports a path directly rather than selecting a discovered entry.

## Implementation Phases

### Phase 1: Classifier

- Add an internal classification result and stable reason constants.
- Parse with `gopkg.in/yaml.v3` and inspect node kinds explicitly.
- Reject empty, malformed, multi-document, duplicate-key, and structurally
  non-matching content.
- Keep the classifier free of filesystem access and Markov subprocess calls so
  it can be tested as a pure unit.

### Phase 2: Discovery Integration

- Apply directory detection and descendant exclusion before standalone parsing.
- Read each remaining YAML candidate once and run the classifier.
- Emit only positive standalone candidates.
- Preserve deterministic path ordering and name derivation.
- Decide how debug-level rejection reasons will be observed without logging one
  line per unrelated YAML file during normal operation.

### Phase 3: Verification

- Extend `internal/projects/git_test.go` with mixed-repository fixtures.
- Add a classifier table test covering positive, negative, malformed, and
  ambiguous cases.
- Verify directory roots are still returned once and their contents are never
  classified independently.
- Verify `.yaml` and `.yml` candidates behave identically.
- Verify direct import still performs authoritative Markov validation.
- Exercise the Projects page against a repository containing both Markov and
  non-Markov YAML and confirm only the expected definitions render.

## Acceptance Criteria

- [x] Standalone discovery parses YAML structure instead of accepting every
      `.yaml` and `.yml` filename.
- [x] The classifier implements the ADR-0004 signature and returns deterministic
      results for the same content.
- [x] Optional Markov sections do not become false requirements.
- [x] Common non-Markov YAML shapes are omitted from project file listings.
- [x] Malformed or non-matching YAML does not fail the whole listing request.
- [x] Filesystem errors retain useful error propagation.
- [x] Directory workflow behavior and descendant exclusion do not regress.
- [x] Listing remains deterministic and does not invoke the Markov binary.
- [x] Import-time Markov validation remains unchanged.
- [x] Backend unit and integration coverage exercises both extensions and mixed
      repositories.
- [x] The Projects page is verified with a representative mixed repository.

## Files Likely Involved

- `internal/projects/git.go`
- `internal/projects/git_test.go`
- `internal/api/projects.go`
- `internal/api/projects_test.go`, if endpoint-level coverage is needed
- `ui/src/pages/Projects.tsx` only if classification metadata is later exposed

## Risks And Mitigations

False negatives from future Markov schema changes:

Keep the classifier isolated, cover known valid shapes with fixtures, and treat
ADR-0004 as the versioned discovery contract. Update the classifier when Markov
introduces another valid single-file form.

False positives from structurally similar YAML:

Require the entrypoint-to-workflow-name relationship and retain authoritative
Markov validation during import.

Large repositories:

Skip directory workflow descendants before reading files, parse each remaining
candidate once, and avoid spawning Markov processes. Measure before adding
concurrency or file-size policy that could make behavior less predictable.

## Deliverables

- Standalone YAML classifier and reasoned classification results.
- Filtered project workflow discovery.
- Backend regression tests and representative mixed-repository fixtures.
- Recorded UI verification evidence in the implementing task or bug file.
