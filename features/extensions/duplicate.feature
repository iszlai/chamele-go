Feature: Duplicate-code extension
  As a developer
  I want duplicate code blocks detected across files
  So that I can refactor copy-pasted logic

  Background:
    Given chamele is configured with default options

  Scenario: A file with one short function still produces a clean result
    Given a C file containing:
      """
      int single(int x) { return x; }
      """
    When I analyze it
    Then 1 function should be detected
