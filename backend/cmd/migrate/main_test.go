package main

import (
	"errors"
	"testing"
)

type fakeMigrationRunner struct {
	downOneCalls int
	downAllCalls int
	steps        []int
	version      uint
	dirty        bool
	err          error
}

func (f *fakeMigrationRunner) DownOne() error {
	f.downOneCalls++
	return f.err
}

func (f *fakeMigrationRunner) DownAll() error {
	f.downAllCalls++
	return f.err
}

func (f *fakeMigrationRunner) Steps(count int) error {
	f.steps = append(f.steps, count)
	return f.err
}

func (f *fakeMigrationRunner) Version() (uint, bool, error) {
	return f.version, f.dirty, f.err
}

func TestRunDownRequiresExplicitAll(t *testing.T) {
	t.Parallel()

	runner := &fakeMigrationRunner{}
	if err := runDown(runner, nil); err != nil {
		t.Fatalf("runDown(single) error = %v", err)
	}
	if runner.downOneCalls != 1 || runner.downAllCalls != 0 {
		t.Fatalf("single down calls = (%d, %d)", runner.downOneCalls, runner.downAllCalls)
	}

	if err := runDown(runner, []string{"--all"}); err != nil {
		t.Fatalf("runDown(all) error = %v", err)
	}
	if runner.downAllCalls != 1 {
		t.Fatalf("down all calls = %d, want 1", runner.downAllCalls)
	}

	if err := runDown(runner, []string{"unexpected"}); err == nil {
		t.Fatal("runDown(unexpected) error = nil")
	}
}

func TestRunStepsRejectsZeroAndPropagatesErrors(t *testing.T) {
	t.Parallel()

	runner := &fakeMigrationRunner{}
	if err := runSteps(runner, []string{"0"}); err == nil {
		t.Fatal("runSteps(0) error = nil")
	}
	if err := runSteps(runner, []string{"2"}); err != nil {
		t.Fatalf("runSteps(2) error = %v", err)
	}
	if len(runner.steps) != 1 || runner.steps[0] != 2 {
		t.Fatalf("steps = %v, want [2]", runner.steps)
	}

	runner.err = errors.New("migration failed")
	if err := runSteps(runner, []string{"-1"}); !errors.Is(err, runner.err) {
		t.Fatalf("runSteps() error = %v, want propagated error", err)
	}
}
