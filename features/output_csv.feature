Feature: CSV output format
  As a CI integrator
  I want a CSV report
  So that downstream tooling can ingest function metrics

  Background:
    Given chamele is configured with default options

  Scenario: A simple function is rendered as one CSV row
    Given a C file containing:
      """
      int add(int a, int b) { return a + b; }
      """
    When I analyze it
    And I render the result as CSV
    Then the output should contain "add"
    And the output should contain "1"
