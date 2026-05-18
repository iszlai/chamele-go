Feature: XML output format
  As a CI integrator
  I want a cppncss-compatible XML report
  So that Jenkins and friends can ingest the results

  Background:
    Given chamele is configured with default options

  Scenario: The XML output mentions the function name
    Given a C file containing:
      """
      int multiply(int a, int b) { return a * b; }
      """
    When I analyze it
    And I render the result as XML
    Then the output should contain "<measure"
    And the output should contain "multiply"
