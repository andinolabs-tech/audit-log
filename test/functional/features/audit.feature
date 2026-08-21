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

  Scenario: Query events within a date range that covers the event
    When I write a minimal audit event for tenant "range-tenant"
    And I query events for tenant "range-tenant" within the last hour
    Then I should receive at least one event

  Scenario: Query events within a date range that excludes the event
    When I write a minimal audit event for tenant "range-tenant"
    And I query events for tenant "range-tenant" for the day before yesterday
    Then I should receive no events

  Scenario: Query events returns the newest first
    When I write a minimal audit event for tenant "order-tenant"
    And I write a minimal audit event for tenant "order-tenant"
    And I query events for tenant "order-tenant" with page size 10
    Then the events should be ordered by date descending
