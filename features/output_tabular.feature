Feature: Tabular output format (default)
  As a developer at the terminal
  I want a human-readable table
  So that I can see complexity numbers at a glance

  Background:
    Given chamele is configured with default options

  Scenario: Tabular output lists the function name and its NLOC
    Given a C file containing:
      """
      int sub(int a, int b) { return a - b; }
      """
    When I analyze it
    And I render the result as tabular
    Then the output should contain "sub"
    And the output should contain "NLOC"
