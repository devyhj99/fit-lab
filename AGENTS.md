# Repository Guidelines

## Repository Structure

- Check the existing directory layout before adding or moving files.
- Keep code grouped by responsibility, following the patterns already used in the repository.
- Place tests and supporting fixtures alongside the code or in the established test locations.
- Do not commit generated artifacts, local data, credentials, or other machine-specific files.

## Development Workflow

- Read the relevant implementation and project documentation before making changes.
- Use the build, formatting, linting, and test commands documented by the project; prefer existing scripts over introducing new ones.
- Keep changes focused, and update documentation or configuration when behavior or setup requirements change.
- Do not add dependencies unless they are needed for the requested change.

## Code and Tests

- Follow the language conventions, formatting, naming, and architectural patterns already established in the codebase.
- Add or update tests when changing behavior, covering expected results and relevant edge cases.
- Use sanitized, minimal fixtures; never put secrets or real personal data in tests or examples.
- Report which checks were run and note any checks that could not be run.

## Security and Data Handling

- Keep secrets and local configuration out of version control; provide safe placeholders in example configuration.
- Validate untrusted input at system boundaries and handle errors explicitly.
- Treat user data as sensitive; collect, log, and retain only what the feature requires.

## Changes and Reviews

- Use clear, concise commit messages that describe the change.
- For a pull request, summarize the motivation and implementation, list relevant validation, and include screenshots when they help review a UI change.
