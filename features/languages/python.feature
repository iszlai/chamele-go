Feature: Python language reader
  As a developer
  I want Python sources analysed correctly
  So that I get accurate metrics for def/async def functions

  Background:
    Given chamele is configured with default options

  Scenario: A simple def function is detected
    Given a Python file containing:
      """
      def hello(name):
          print(name)
      """
    When I analyze it
    Then 1 function should be detected
    And the function "hello" should have CCN 1
    And the function "hello" should have 1 parameter

  Scenario: if/elif each add 1 to CCN
    Given a Python file containing:
      """
      def cat(x):
          if x > 0:
              return 'pos'
          elif x < 0:
              return 'neg'
          else:
              return 'zero'
      """
    When I analyze it
    Then the function "cat" should have CCN 3
