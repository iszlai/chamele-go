Feature: Forgiveness directives
  As a developer
  I want #lizard forgive comments to suppress warnings for individual functions
  So that I can acknowledge known complexity without cluttering reports

  Background:
    Given chamele is configured with default options

  Scenario: #lizard forgive suppresses the current function's CCN warning
    Given a Go file containing:
      """
      package x
      // #lizard forgive
      func complex() {
          if a {}
          if b {}
      }
      """
    When I analyze it
    Then no functions should be detected
