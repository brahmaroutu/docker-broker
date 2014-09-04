package brokerapi_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "testing"
)

func TestBrokerapi(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Brokerapi Suite")
}
