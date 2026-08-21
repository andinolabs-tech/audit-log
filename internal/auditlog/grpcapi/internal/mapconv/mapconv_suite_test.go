package mapconv_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMapconv(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "gRPC API Mapconv Suite")
}
