Feature: JavaScript language reader
  As a developer
  I want JavaScript sources analysed correctly
  So that I get accurate metrics for function declarations and expressions

  Background:
    Given chamele is configured with default options

  Scenario: A named function declaration is detected
    Given a JavaScript file containing:
      """
      function greet(name) { return "hello, " + name; }
      """
    When I analyze it
    Then 1 function should be detected
    And the function "greet" should have CCN 1
    And the function "greet" should have 1 parameter
