Feature: Core analysis behaviour
  As a developer using chamele as a library
  I want NLOC, CCN, token count, and parameter count computed from source
  So that I can measure complexity of any supported language file

  Background:
    Given chamele is configured with default options

  Scenario: A Go function with no branches has CCN 1
    Given a Go file containing:
      """
      package x
      func hello() { fmt.Println("hi") }
      """
    When I analyze it
    Then the function "hello" should have CCN 1
    And the function "hello" should have 0 parameters

  Scenario Outline: Common Go control-flow constructs each add 1 to CCN
    Given a Go file containing:
      """
      package x
      func f(a int) bool { <body> }
      """
    When I analyze it
    Then the function "f" should have CCN <ccn>

    Examples:
      | body                              | ccn |
      | if a > 0 { return true }           |  2  |
      | for i := 0; i < a; i++ { _ = i }  |  2  |
      | return a > 0 && a < 10            |  2  |

  Scenario: Nested functions both appear in results
    Given a Go file containing:
      """
      package x
      func outer() {
          inner := func() {}
          _ = inner
      }
      """
    When I analyze it
    Then 2 functions should be detected

  Scenario: A Python function is detected
    Given a Python file containing:
      """
      def greet(name):
          print(name)
      """
    When I analyze it
    Then 1 function should be detected
    And the function "greet" should have 1 parameter

  Scenario: A C function is detected
    Given a C file containing:
      """
      int add(int a, int b) { return a + b; }
      """
    When I analyze it
    Then 1 function should be detected
    And the function "add" should have 2 parameters

  Scenario: A Java class method is detected
    Given a Java file containing:
      """
      class Calc {
          int add(int a, int b) { return a + b; }
      }
      """
    When I analyze it
    Then 1 function should be detected
    And the function "Calc::add" should have CCN 1
