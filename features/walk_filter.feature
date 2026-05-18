Feature: File walking and filtering
  As a developer running chamele over a directory
  I want unsupported files, excluded paths, and duplicates skipped
  So that only relevant source files are analysed

  Background:
    Given chamele is configured with default options

  Scenario: Only files matching a known language reader are analysed
    Given a directory with these files:
      | filename  | content                       |
      | a.go      | package x;func f() {}         |
      | README.md | # not source                  |
    When I analyze the directory
    Then 1 file should be reported

  Scenario: Exact-duplicate files are deduplicated by content hash
    Given a directory with these files:
      | filename  | content                       |
      | a.go      | package x;func f() {}         |
      | b.go      | package x;func f() {}         |
    When I analyze the directory
    Then 1 file should be reported
