Feature: CLI exit code semantics
  As a CI script author
  I want the binary's exit status to reflect whether thresholds were exceeded
  So that build pipelines can gate on complexity violations

  Background:
    Given chamele is configured with default options

  Scenario: Clean code yields a function with CCN 1 (a non-warning baseline)
    Given a C file containing:
      """
      int identity(int x) { return x; }
      """
    When I analyze it
    Then the function "identity" should have CCN 1
