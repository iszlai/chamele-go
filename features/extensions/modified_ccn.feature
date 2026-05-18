Feature: Modified CCN extension
  As a developer
  I want switch/case to count as a single decision under modified CCN
  So that the metric better reflects readability of branch-tables

  Background:
    Given chamele is configured with default options

  Scenario: An if/else still increments CCN normally
    Given a C file containing:
      """
      int sign(int x) {
          if (x > 0) return 1;
          else return -1;
      }
      """
    When I analyze it
    Then the function "sign" should have CCN 2
