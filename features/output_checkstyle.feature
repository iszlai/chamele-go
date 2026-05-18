Feature: Checkstyle XML output format
  As a CI integrator
  I want Checkstyle-compatible XML
  So that PR comment tooling that understands Checkstyle can ingest it

  Background:
    Given chamele is configured with default options

  Scenario: A high-CCN function is flagged in Checkstyle XML
    Given a C file containing:
      """
      int big(int x) {
          if (x > 0) { if (x > 1) { if (x > 2) {
            if (x > 3) { if (x > 4) { if (x > 5) {
              if (x > 6) { if (x > 7) { if (x > 8) {
                if (x > 9) { if (x > 10) { if (x > 11) {
                  if (x > 12) { if (x > 13) { if (x > 14) {
                      return 1;
                  } } } } } } } } } } } } } } }
          return 0;
      }
      """
    When I analyze it
    And I render the result as checkstyle
    Then the output should contain "checkstyle"
    And the output should contain "big"
