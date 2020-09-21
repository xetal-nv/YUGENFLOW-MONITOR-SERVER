# Xetal Flow Monitoring GateServer

Copyright Xetal @ 2019  
Author: F. Pessolano  

**TO BE FULLY REDONE, VERSION BREAKS BACK COMPATIBILITY**

**REQUIREMENTS**  
GO 1.14 or newer  
64-bit architecture  
Golang Packages (to be revised):
 - go.etcd.io/bbolt/  
 - xetal.ddns.net/supportservices  
 - github.com/mongodb/mongo  
 - github.com/allegro/bigcache  
 - github.com/fpessolano/mlogger  (>=0.2.1)  
 - github.com/gorilla/mux  
 - gopkg.in/ini.v1  
 
External data:  
  - tbd
  
External services:
  - tbd  

**API:(TBD)**  
HTTPSPORTS[1]/ks -> if enabled, kills the server  
HTTPSPORTS[1]/dvl -> if enabled, shows latest developer log 
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
HTTPSPORTS[1]/presence?space=z?analysis=y?start=x0?end=x1 -> samples of type presence from space z and analysis y from timestamp x0 to timestamp x1, the value is set to 2 when activity is detected in the period and the value is equal to the number of detection at the end of the period  
HTTPSPORTS[1]/command?cmd=x?id=y?mac=w?val=z -> execute command x on sensor y or w with data (if necessary) z when z is an array. If both y and w are specified it returns error    
HTTPSPORTS[1]/command?cmd=macid?id=y?val=z -> assigns the id y to device with mac z of the device has currently id 0xff, mac must be passed given as a sequence if hex values like 1a:62:63:ef:32:36  
HTTPSPORTS[1]/command?cmd=list -> lists all available commands  
HTTPSPORTS[1]/command?pin=xyz -> sends debug pin xyz, answer true is accepted, nothing otherwise  
HTTPSPORTS[1]/dbs/retrieve/samples -> retrieve sample data from .recoverysamples if dbsupdate is set   
HTTPSPORTS[1]/dbs/retrieve/presence -> retrieve sample data from .recoverypresence if dbsupdate is set   

  
NOTE: values in val are specified as x,y,n,..   
NOTE: series API supports types: sample, entry (debug mode only). Data for the current day are only available in debug mode  

**SYSTEM VARIABLES:**  
n/a  

**CONFIGURATION:**  
See gateserver.ini and configuration.ini  file for configuration example  

**COMMAND LINE OPTIONS:**  
-debug                  : enables debug mode (to be done)  
-db path                : set database path (to be done)  
-dc path                : set disk cache path (to be done)  
-user username          : set username (to be done)  
-pwd password           : set password (to be done)      
-echo                   : enter the echo mode (to be done)  
-cdelay int             : specifies the maximum delay for recovery data usage(to be done)  
-dmode Int              : specifies an execution mode (0: default, 1: full test, 2: short test) (to be done)  
-dumpentry              : forces all entry values/activity to be written in a log file for debug (to be done)  
-st string              : set start time, time specified as HH:MM (to be done)  
-eeprom                 : enables refresh of device eeprom at every connection (to be done)  
-nosample               : disable automatic check for database recovery (to be done)  
-dbsupdate              : enable DBS integrity check HTTP API (to be done)  
-tdl int                : TCP read deadline in hours (default 24) (to be done)  

**INSTALLATION**  
Executable file: gateserver(.exe)  
Configuration files: gateserver.ini, configuration.ini  
Resource folders: 

**TO BE DONE (in priority order)**  
 - everything


