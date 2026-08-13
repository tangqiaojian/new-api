package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadStartupModeRejectsMigrationOnlySlave(t *testing.T) {
	t.Setenv("MIGRATION_ONLY", "true")
	t.Setenv("NODE_TYPE", "slave")

	_, err := readStartupMode()

	require.EqualError(t, err, "MIGRATION_ONLY=true cannot be combined with NODE_TYPE=slave: slave nodes skip database migrations")
}

func TestReadStartupModeNormalizesNodeType(t *testing.T) {
	t.Setenv("MIGRATION_ONLY", "true")
	t.Setenv("NODE_TYPE", " Slave ")

	_, err := readStartupMode()

	require.EqualError(t, err, "MIGRATION_ONLY=true cannot be combined with NODE_TYPE=slave: slave nodes skip database migrations")
}

func TestReadStartupModeAllowsSupportedCombinations(t *testing.T) {
	tests := []struct {
		name          string
		migrationOnly string
		nodeType      string
		wantMigration bool
	}{
		{name: "regular master", migrationOnly: "false", nodeType: "master", wantMigration: false},
		{name: "regular slave", migrationOnly: "false", nodeType: "slave", wantMigration: false},
		{name: "migration master", migrationOnly: "true", nodeType: "master", wantMigration: true},
		{name: "migration default node", migrationOnly: "true", nodeType: "", wantMigration: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("MIGRATION_ONLY", test.migrationOnly)
			t.Setenv("NODE_TYPE", test.nodeType)

			mode, err := readStartupMode()

			require.NoError(t, err)
			assert.Equal(t, test.wantMigration, mode.migrationOnly)
		})
	}
}

func TestStartupModeInitializeRuntime(t *testing.T) {
	t.Run("migration only skips runtime initializer", func(t *testing.T) {
		called := false
		err := startupMode{migrationOnly: true}.initializeRuntime(func() error {
			called = true
			return errors.New("runtime initializer must not run")
		})

		require.NoError(t, err)
		assert.False(t, called)
	})

	t.Run("regular startup runs runtime initializer", func(t *testing.T) {
		called := false
		wantErr := errors.New("runtime failed")
		err := startupMode{}.initializeRuntime(func() error {
			called = true
			return wantErr
		})

		require.ErrorIs(t, err, wantErr)
		assert.True(t, called)
	})
}
