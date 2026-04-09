package database_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"audit-log/internal/auditlog/persistence"
	"audit-log/internal/infra/database"
)

var _ = Describe("OpenInMemory", func() {
	It("returns a non-nil *gorm.DB without error", func() {
		db, err := database.OpenInMemory()
		Expect(err).NotTo(HaveOccurred())
		Expect(db).NotTo(BeNil())
	})

	It("returns a connection that supports AutoMigrateModel", func() {
		db, err := database.OpenInMemory()
		Expect(err).NotTo(HaveOccurred())
		Expect(persistence.AutoMigrateModel(db)).To(Succeed())
	})
})
