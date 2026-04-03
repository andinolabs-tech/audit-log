package jsonpatch_test

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"audit-log/internal/infra/jsonpatch"
)

func TestJSONPatch(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "JSONPatch Suite")
}

var _ = Describe("DiffMaps", func() {
	It("returns RFC 6902 operations as a map with an operations slice", func() {
		before := map[string]any{"name": "a", "count": float64(1)}
		after := map[string]any{"name": "b", "count": float64(1)}
		diff, err := jsonpatch.DiffMaps(before, after)
		Expect(err).NotTo(HaveOccurred())
		ops, ok := diff["operations"].([]any)
		Expect(ok).To(BeTrue(), "expected operations key with JSON array unmarshaled as []any")
		Expect(ops).NotTo(BeEmpty())
		// At least one replace for name
		raw, err := json.Marshal(ops)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(raw)).To(ContainSubstring(`"op"`))
		Expect(string(raw)).To(ContainSubstring(`/name`))
	})

	It("returns empty operations when before and after are equal", func() {
		m := map[string]any{"x": true}
		diff, err := jsonpatch.DiffMaps(m, m)
		Expect(err).NotTo(HaveOccurred())
		ops := diff["operations"].([]any)
		Expect(ops).To(BeEmpty())
	})

	It("returns error when marshaling fails for non-JSON values", func() {
		before := map[string]any{"c": make(chan int)}
		after := map[string]any{"c": make(chan int)}
		_, err := jsonpatch.DiffMaps(before, after)
		Expect(err).To(HaveOccurred())
	})
})
