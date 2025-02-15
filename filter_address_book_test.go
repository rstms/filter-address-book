package main

import (
	"fmt"
	"github.com/rstms/rspamd-classes/classes"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestVersion(t *testing.T) {
	SpamClasses, err := classes.New("")
	require.Nil(t, err)
	require.NotEmpty(t, classes.Version)
	require.NotNil(t, SpamClasses)
	fmt.Printf("Version=v%s  rspamd_classes.Version=v%s\n", Version, classes.Version)
}
