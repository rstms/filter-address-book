package main

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

const TestUser = "test@mailcapsule.io"
const TestAddress = "test@bootnotice.com"
const TestAddressWithBrackets = "test address <test@bootnotice.com>"

func TestBrackets(t *testing.T) {

	err := Configure("testdata/config.yml")
	require.Nil(t, err)
	client := NewFilterControlClient()
	books, err := client.ScanAddressBooks(TestUser, TestAddressWithBrackets)
	require.Nil(t, err)
	for _, book := range books {
		fmt.Printf("%s\n", book)
	}
}

func TestNoBrackets(t *testing.T) {

	err := Configure("testdata/config.yml")
	require.Nil(t, err)
	client := NewFilterControlClient()
	books, err := client.ScanAddressBooks(TestUser, TestAddress)
	require.Nil(t, err)
	for _, book := range books {
		fmt.Printf("%s\n", book)
	}
}
