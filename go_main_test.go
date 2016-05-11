package main

import (
  "testing"
  . "github.com/onsi/ginkgo"
  . "github.com/onsi/gomega"
)

func TestGo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Go Test")
}

var _ = BeforeSuite(func() {
  SetTestRequestHandler()
})
