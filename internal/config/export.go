package config

// The frontend mirrors Defaults() and Limits() through the generated module
// at frontend/src/shared/generated/constants.ts. Whenever any value here
// changes, run `task gen` to regenerate the TS file. CI/pre-push enforces
// drift detection.

//go:generate go run ../../cmd/gentsconst
