Feature: Exit count extension
  As a developer
  I want each function's exit-point count tracked
  So that I can flag functions with too many early returns

  Background:
    Given chamele is configured with default options

  Scenario: A function with a single return is still analysed
    Given a C file containing:
      """
      int once(int x) { return x; }
      """
    When I analyze it
    Then the function "once" should have CCN 1
