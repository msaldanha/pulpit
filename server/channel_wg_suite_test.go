package server

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestChannelWaitGroup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ChannelWaitGroup Suite")
}
