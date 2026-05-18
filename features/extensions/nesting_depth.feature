Feature: Nesting depth (ND) extension
  As a developer
  I want the maximum nesting depth per function reported
  So that deeply-nested code is surfaced

  Background:
    Given chamele is configured with default options

  Scenario: A flat function has nesting depth at most 1
    Given a C file containing:
      """
      int flat(int x) { return x + 1; }
      """
    When I analyze it
    Then the function "flat" should have CCN 1
