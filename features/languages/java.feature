Feature: Java language reader
  As a developer
  I want Java sources analysed correctly
  So that I get accurate metrics for class methods

  Background:
    Given chamele is configured with default options

  Scenario: A class method is detected with its qualified name
    Given a Java file containing:
      """
      class Calc {
          int add(int a, int b) { return a + b; }
      }
      """
    When I analyze it
    Then 1 function should be detected
    And the function "Calc::add" should have CCN 1
    And the function "Calc::add" should have 2 parameters
