# Changelog
All notable changes to this project will be documented in this file.

## [2.2.1]
### Changed
- code cleaning  

## [2.1.1]
### Changed
- removed sources of context leaks  

### Added
- periodic unmarking of malicious entries (in progress)  

## [2.0.1]
### Changed
- fixed a bug preventing the counter reset to take place without the sensors triggering the counter itself  

## [2.0.0]
### This new branch is a complete re-write of the server for YugenFlow 2v support   

### Added
 - Possibility to export data via external script  
 - Option to remove all logs  
 - Build constraints  

### Changed
 - Server architecture has been changed  
 - Database is no longer based on MongoDB and not badger  
 - Malicious attack check has been simplified, more complex checks will be added later    
 - Several bugs have been removed  
 - Improved stress test  

