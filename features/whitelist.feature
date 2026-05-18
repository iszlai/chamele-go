Feature: Whitelist suppression
  As a developer
  I want functions listed in whitelizard.txt to be suppressed from warnings
  So that pre-existing complexity hot-spots can be tracked without noise

  Background:
    Given chamele is configured with default options

  Scenario: A function present in the whitelist is excluded from warnings
    Given a C file containing:
      """
      int complex(int x) {
          if (x > 0) {
              if (x > 1) {
                  if (x > 2) {
                      if (x > 3) {
                          return 4;
                      }
                  }
              }
          }
          return 0;
      }
      """
    When I analyze it
    Then 1 function should be detected
