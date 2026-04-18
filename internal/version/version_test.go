package version_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"audit-log/internal/version"
)

var _ = Describe("String", func() {
	It("trims space and returns injected version", func() {
		old := version.Version
		DeferCleanup(func() { version.Version = old })
		version.Version = "  v1.2.3  "
		Expect(version.String()).To(Equal("v1.2.3"))
	})

	It("returns a non-empty value when Version is empty", func() {
		old := version.Version
		DeferCleanup(func() { version.Version = old })
		version.Version = ""
		Expect(version.String()).NotTo(BeEmpty())
	})
})
