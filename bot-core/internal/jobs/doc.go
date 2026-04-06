// Package jobs owns background fanout and progress-tracked work.
// The jobs table is already part of the canonical schema; higher-level
// fanout handlers are deferred until Phase 3.
package jobs
