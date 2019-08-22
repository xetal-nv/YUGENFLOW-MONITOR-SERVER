# Changelog
All notable changes to this project will be documented in this file.

## [0.8.0]
### Added
 - Debug webapp at dbg.html
 - Pin controlled access to entry data, uses API command?pin= and cen be entered also form debug webapp
 - Added graphs for current values, including export of graphs
 
### Changed
 - Split configuration file in system configuration (.systemenv) and installation configutation (.env or other). Back compatible
 - Reporting average use trunc.round instead of round
 - API 'cmd' renamed 'command', both path valid for back compatibility. Path 'cmd' will be removed in 0.9.0
 - Improved speed and network usage of real time values in the webapp by using the SPACE API instead of the REGISTER one
 - Web app will not provide data for the current day, web app for debug does
 - Added support for absence of plan, a message is displayed instead
 - Disabling current from servers command option only disables it for reporting
 - Increased default timeout values for API on both server and web app (to accomodate issues with the debug interface and current samples)  
 - Web app interface has been cleaned and adapted to accommodate graphs

## [0.7.0]
### Added
 - Configuration RTWINDOW defining a reporting window for real time data. It overrides ANALYSISWINDOW for reporting only  
 - Overview report mode with configurable template
 - Option REPSHOW to enable specific reports  
 - Option RTSHOW to enable specific real time values   
 - Added a new API for presence detection including configuration options, use "presence" as type in the series API  
 - Added dumpentry command line option for debug purposed  
 - Added check on compulsory configuration variables  
 - Add recovery for detection presence datapath  
 - Added support for point reporting weekly and period averages 
 - Added support for fall back analysis of data if presence data not available yet for reporting  
 - Added bypass.js for forcing JS changes without restart of the server and for sing new JS on old server verions  
 - Added optimised DB driver for presence detection (IP)  
 - Changelog file
 - Period averages extended to all reported values
 - Added calendar week numbers to report
 - Added activity reporting default (set in bypass.js)
 
### Changed
 - Consolidated all variables for JS, which are generated dynamically only once, in one file def.js  
 - Solved issues with european copies of Excel  
 - In case overview is the only report available, the selector is now hidden  
 - Resolved bug that report action wheel does not disappear with no valid data  
 - .recovery file renamed .recoveryavg tp specify the datapath it belongs to  
 - Improved resistance of malicious check errors due to channel misalignment  
 - Changed recovery policy from always showing a sample in the webapp if the counter if not too old, independently form how old the sample average is  

### Removed
 - Removed redundant check on report generation  
 - Removed declaration of DBS reset channels, they are just placeholders.  
 - Removed possible race at DBS creation  
 - Removed CRC control on on reset 

 
## [v0.6.0] 

### BUG WARNING
 - This version only works with option '-norst' and ANALYSISWINDOW must be defined  

### Added
 - Added ANALYSISWINDOW configuration option that synchronise analysis and define "working hours"  
 - Added delay server start with command option 'st'  
 - Added additional CRC check in start-up reset  
 - Added possibility to declare maximum value for each space  
 - Added support for ANALYSISWINDOW in JS  
 - Added option MULTICYCLEDAYSONLY to force multicycles to be only multiple of days  
 
### Changed
 - Reporting on current samples is now to be enables by command line with -repcon  
 - Resolved minor bug that would skip the first minute in any time schedule provided in the configuration file  
 - Some cosmetic changes to the code and interface  
 - Fixed bug preventing CLOSURE_ from working always
 - Improved averaging algorithm with edge cases of missing samples/averages near the analysis period end  

### Removed
 - Removed CMODE 3  

## [0.5.1]

### Added

### Changed
 - !!! Renamed SAVEWINDOW to ANALYSISPERIOD in configuration file !!!  
 - Modified the averaging algorithm to support forthcoming ANALYSISWINDOW implementation  

### Removed
 - Removed CSTAT option from configuration file as not useful  


## [v0.5.0]

### Added
- Added start-up background reset of sensors (first connection from server start only)  
- In case of server crash, the data is stored in a file called .recovery and used if no older than what specified with the  command line option cdelay (std 30s)  
- Added command line options ks (kill switch), norst (skip start-up reset of sensors), cdelay (see recovery file) and nomal (disable malicious attack monitoring)  

### Changed
- Improved algorithm for entry sampling in case of failure of one of the two sensors  
- Solved memory leaks on several mutex elements  
- Resolved issue with web app reporting illegal values when in real time monitoring  
- Resolved issue causing negative entries not to be displayed in the real time web app  
- Reduced network load when preparing the report via web app including removal of entry values in the report  
- In case of network issues, the web app will try again a given number of times before reporting an issue to the user  

### Removed
- Removed bug in counter resets that prevented entry values to be reset  
- Removed averaging of entry values, cumulative are given instead  
- Removal of interpolation in reporting sue to large errors it introduces  
