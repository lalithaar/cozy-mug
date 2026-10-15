# 0022 — Documentation style and task-first ordering

- Status: firm
- Date: 2026-10-15
- Supersedes: none

## Opinion

cozy-mug documentation follows the Google developer documentation style and a
task-first page order.

- **Voice and tense:** active voice, present tense, second person for
  instructions.
- **Structure:** sentence-case headings; short paragraphs; code snippets that
  can be copied and run.
- **Task sequence:** every guide leads with a minimal working recipe (numbered
  steps), then what the framework handles, then ownership/responsibility, then
  production best practices and pitfalls.
- **Default ordering for new pages:**
  1. Quick start / minimal recipe
  2. What cozy-mug handles
  3. What you still own
  4. Best practices
  5. Common pitfalls
- Private or advanced branches come after the common path so they do not
  interrupt the recipe.

## Because

- Readers look up documentation to complete a task first; the recipe must
  appear before theory.
- A consistent order makes reference predictable across pages.
- The Google style guide keeps wording direct and lowers ambiguity.

## Apply when

Writing or revising any user-facing documentation, including Starlight pages,
README examples, and idioms chapters that serve as user docs.

## Deviate / ask when

A page is purely conceptual (no task recipe) or serves a reference-only role
such as generated API tables; the task-first order does not apply there.