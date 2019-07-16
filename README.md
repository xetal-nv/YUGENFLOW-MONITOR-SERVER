# gateserver

Copyright Xetal @ 2019  
Author: F. Pessolano  

**Requirements:**  
GO 1.12 or compatible  
64-bit architecture  
Badger version 1.5.3  
Need to be cloned to go/src for compilation  

**List API:**  
HTTPSPORTS[1]/ks -> if enabled, kills the server  
HTTPSPORTS[1]/dvl -> latest developer log (IF ENABLED via -dvl)  
HTTPSPORTS[1]/asys -> information on all current analyses  
HTTPSPORTS[1]/info -> installation information  
HTTPSPORTS[1]/pending -> list all devices pending for connection approval (only current connections)    
HTTPSPORTS[1]/active -> list all valid connected devices  
HTTPSPORTS[1]/und -> list all connected devices that are not used in the installation  
HTTPSPORTS[1]/udef -> list all devices with undefined id 0xff that have been connected  
HTTPSPORTS[1]/udef/active -> list all connected devices with initial id 0xff  
HTTPSPORTS[1]/udef/notactive -> list all not connected devices with initial id 0xff  
HTTPSPORTS[1]/udef/defined -> list all defined devices with initial id 0xff  
HTTPSPORTS[1]/udef/undefined -> list all not yet defined devices with initial id 0xff  
HTTPSPORTS[1]/x/y/z -> actual value for data x in space y on averaging z  
HTTPSPORTS[1]/series?last=x?type=y?space=z?analysis=y -> last x samples of type y from space z and analysis y  
HTTPSPORTS[1]/series?type=y?space=z?analysis=y?start=x0?end=x1 -> samples of type y from space z and analysis y from timestamp x0 to timestamp x1  
HTTPSPORTS[1]/cmd?cmd=x?id=y?chan=w?val=z -> execute command x on sensor y or w with data (if necessary) z when z is an array. If both y and w are specified it returns error    
HTTPSPORTS[1]/cmd?cmd=macid?id=y?val=z -> assigns the id y to device with mac z of the device has currently id 0xff, mac must be passed given as a sequence if hex values like 1a:62:63:ef:32:36  
HTTPSPORTS[1]/cmd?list -> lists all available commands  
  
NOTE: values in val are specified as x,y,n,..   
NOTE: all commands need to be fully tested via mac and id  

**List HTTP pages:**  
HTTPSPORTS[0]/ -> webapp

**SVG convention:**  
Elements triggering data from a entry need to have as ID the entry id as from the configuration file  
Elements triggering data from the full counter need to have as ID the space name  
Two classes need to be defined, st1 for unselected trigger and st2 for selected trigger  
If the server does no have a svg for a given space, the space will be ignored  

**SYSTEM VARIABLES:**  
GATESERVER is set to the application folder  

**CONFIGURATION:**  
See .env file for configuration example

**COMMAND LINE OPTIONS:**  
-dbs Path : specifies path where to store the database, './DBS' used if not specified  
-cdelay int : specifies the maximum delay for recovery data usage  
-debug Int : specifies a debug mode (0: off, -1: short flow noalgo, 1: verbose, 2: verbose no algo, 3: verbose no dbs, 4: verbose no algo no dbs)  
-dellogs : delete all existing logs  
-dmode Int : specifies an execution mode (0: default, 1: full test, 2: short test)  
-dvl : activate dvl  
-env String : specifies the configuration file, uses .env if not specified  
-ks : enable killswitch API  
-nomal : disable malicious attack control  
-norst : disable start-up device reset  
-repcon : enables current reporting in JS
-ri Int : set log ri  
-rs Int64 : set log rs  
-st string : set start time, time specified as HH:MM

**CHANGELOG TO v0.6.0:**  

 - Added ANALYSISWINDOW configuration option that synchronise analysis and define "working hours"  
 - Added delay server start with command option 'st'  
 - Added additional CRC check in start-up reset  
 - Added possibility to declare maximum value for each space  
 - Reporting on current samples is now to be enables by command line with -repcon  
 - Resolved minor bug that would skip the first minute in any time schedule provided in the configuration file  
 - Removed CMODE 3  
 - Improved averaging algorithm with edge cases of missing samples/averages near the analysis period end  
 - Added support for ANALYSISWINDOW in JS  
 - Some cosmetic changes to the code and interface  
 - Fixed bug preventing CLOSURE_ from working always
 - Code cleaning (to be done)
 - Added option MULTICYCLEDAYSONLY to force multicycles to be only multiple of days  

**CHANGELOG TO v0.5.1:**  

 - !!! Renamed SAVEWINDOW to ANALYSISPERIOD in configuration file !!!  
 - Removed CSTAT option from configuration file as not useful  
 - Modified the averaging algorithm to support forthcoming ANALYSISWINDOW implementation  


**CHANGELOG TO v0.5.0:**  

- Improved algorithm for entry sampling in case of failure of one of the two sensors  
- Removed bug in counter resets that prevented entry values to be reset  
- Solved memory leaks on several mutex elements  
- Added start-up background reset of sensors (first connection from server start only)  
- In case of server crash, the data is stored in a file called .recovery and used if no older than what specified with the  command line option cdelay (std 30s)  
- Removed averaging of entry values, cumulative are given instead  
- Added command line options ks (kill switch), norst (skip start-up reset of sensors), cdelay (see recovery file) and nomal (disable malicious attack monitoring)  
- Resolved issue with web app reporting illegal values when in real time monitoring  
- Resolved issue causing negative entries not to be displayed in the real time web app  
- Removal of interpolation in reporting sue to large errors it introduces  
- Reduced network load when preparing the report via web app including removal of entry values in the report  
- In case of network issues, the web app will try again a given number of times before reporting an issue to the user  

