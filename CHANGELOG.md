# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [0.3.0] - 2025-11-27
### Changed
- `render_diff_views` now only generates images for views with actual architectural changes
- CLI outputs `ARCHLENS_CHANGED_VIEWS=<view1>,<view2>,...` to indicate which views have changes

## [0.2.9] - 2025-06-23
### Bugfix
- Default view shows top level packages not all the packages in the system

## [0.2.8] - 2025-06-23
### Bugfix
- Fixed package depth functionality that was not working properly: when depth is 2 the view should not show nodes at depth 1 or 3! (with Sebastian Cloos Hylander)