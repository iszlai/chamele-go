Feature: Fan-in / fan-out (io) extension
  As a developer
  I want fan-in and fan-out counted across files
  So that I can spot highly-connected procedures

  Background:
    Given chamele is configured with default options

  Scenario: A self-contained function has CCN 1 even with io extension active
    Given a C file containing:
      """
      int isolated(int x) { return x * 2; }
      """
    When I analyze it
    Then the function "isolated" should have CCN 1
