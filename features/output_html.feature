Feature: HTML output format
  As a code-review consumer
  I want a browseable HTML report
  So that I can spot complexity hot-spots visually

  Background:
    Given chamele is configured with default options

  Scenario: HTML output contains an <html> tag and the function name
    Given a C file containing:
      """
      int sq(int a) { return a * a; }
      """
    When I analyze it
    And I render the result as HTML
    Then the output should contain "<html"
    And the output should contain "sq"
