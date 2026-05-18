Feature: C language reader
  As a developer
  I want C sources analysed correctly
  So that I get accurate metrics for C functions

  Background:
    Given chamele is configured with default options

  Scenario: A trivial C function has CCN 1
    Given a C file containing:
      """
      int add(int a, int b) { return a + b; }
      """
    When I analyze it
    Then the function "add" should have CCN 1
    And the function "add" should have 2 parameters

  Scenario: while loop adds 1 to CCN
    Given a C file containing:
      """
      int countdown(int n) {
          while (n > 0) n--;
          return n;
      }
      """
    When I analyze it
    Then the function "countdown" should have CCN 2
