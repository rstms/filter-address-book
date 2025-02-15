package main

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

const TestUser = "test@mailcapsule.io"
const TestAddress = "test@bootnotice.com"

func TestVersion(t *testing.T) {

	err := Configure("testdata/config.yml")
	require.Nil(t, err)
	client := NewFilterControlClient()
	books, err := client.ScanAddressBooks(TestUser, TestAddress)
	require.Nil(t, err)
	for _, book := range books {
		fmt.Printf("%s\n", book)
	}
}
