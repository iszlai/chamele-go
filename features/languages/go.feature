Feature: Go language reader
  As a developer
  I want Go source analysed correctly
  So that I get accurate CCN and parameter counts for Go functions

  Background:
    Given chamele is configured with default options

  Scenario: Simple named function
    Given a Go file containing:
      """
      package x
      func sayHello() {}
      """
    When I analyze it
    Then 1 function should be detected
    And the function "sayHello" should have CCN 1
    And the function "sayHello" should have 0 parameters

  Scenario: Method with receiver
    Given a Go file containing:
      """
      package x
      func (s *S) Foo() {}
      """
    When I analyze it
    Then 1 function should be detected
    And the function "Foo" should have CCN 1

  Scenario: Type definition is not a function
    Given a Go file containing:
      """
      package x
      type MyInterface interface { Method() int }
      """
    When I analyze it
    Then no functions should be detected

  Scenario: Closure inside a function is counted separately
    Given a Go file containing:
      """
      package x
      func outer() {
          f := func() {}
          _ = f
      }
      """
    When I analyze it
    Then 2 functions should be detected

  Scenario: Generic function
    Given a Go file containing:
      """
      package x
      func Map[T, U any](s []T, f func(T) U) []U { return nil }
      """
    When I analyze it
    Then 1 function should be detected
    And the function "Map" should have CCN 1
