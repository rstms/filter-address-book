package main

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

const TestUser = "test@mailcapsule.io"
const TestAddress = "test@bootnotice.com"

func TestVersion(t *testing.T) {

	err := Configure()
	require.Nil(t, err)
	api, err := MAB()
	require.Nil(t, err)
	books, err := ScanAddressBooks(api, TestUser, TestAddress)
	require.Nil(t, err)
	for _, book := range books {
		fmt.Printf("%s\n", book)
	}
}
