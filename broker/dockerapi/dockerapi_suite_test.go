package dockerapi_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "testing"
)

func TestDockerapi(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Dockerapi Suite")
}
