Feature: Audit log gRPC API
  Exercise WriteEvent, GetEvent, and QueryEvents against a running server.

  Scenario: Write and retrieve an event
    When I write a minimal audit event
    Then the write response should include a generated id
    When I get that event by id
    Then the event tenant should be "func-test-tenant"

  Scenario: Query events with filters
    When I write a minimal audit event for tenant "query-tenant"
    And I query events for tenant "query-tenant" with page size 10
    Then I should receive at least one event
