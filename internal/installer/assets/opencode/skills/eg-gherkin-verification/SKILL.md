---
name: eg-gherkin-verification
description: "Trigger: Gherkin, GIVEN WHEN THEN, scenario verification, feature files, BDD. Verify OpenSpec scenarios are observable, valid, and test-traced."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Gherkin Verification

## Activation Contract

Use in QA to verify spec scenarios, and in planning to sanity-check them.

## Hard Rules

- Every scenario is observable: THEN states a result a user or system boundary can show, never an internal detail.
- GIVEN/WHEN/THEN is well-formed: GIVEN context, WHEN a single action, THEN the observable outcome.
- Every requirement has at least one scenario; every scenario traces to test evidence — an automated check or an exploratory charter step.
- Positive, negative, boundary, and failure coverage exists for each behavior.

## Executable Gherkin

Create `.feature` files only when:

1. The repository already runs a BDD runner (Cucumber, Godog, pytest-bdd, etc.), or
2. The engineer explicitly chooses executable Gherkin.

Otherwise scenarios live in OpenSpec specs and map to regular tests by name.

## Output Contract

Report per requirement: scenarios found, validity, and the evidence mapping each one. Untraced or unobservable scenarios are findings.
