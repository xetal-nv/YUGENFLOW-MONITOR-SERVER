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
 - github.com/fpessolano/mlogger  (>=0.2.1)  
 - github.com/gorilla/mux  
 - gopkg.in/ini.v1  
 
External data:  
  - tbd
  
External services:
  - mongoDB database  

**API:** 
/info                                   -> installation information  
/connected                              -> list all connected device which have not been marked invalid and report if they are active or not  
/invalid                                -> list all connected device which have  been marked invalid and report if the invalidity timestamp  
/measurements                           -> returns the definition of all active measurement  
/latestdata                                   -> return latest measurements for all spaces  
/latestdata/{name}                      -> return latest data for space {name}   
/latestdata/{[name0, name1, ...]}       -> return latest data for spaces [name0, name1, ...]  
/reference?n                            -> return the last n reference measurements for all spaces  
/reference/{name}?n                     -> return oly the last n reference measurements for space {name}   
/reference/{[name0, ...]}?n             -> return the last n reference measurements for spaces [name0, name1, ...]  
/delta?n                                -> return the last n real data for all spaces  
/delta/{name}?n                         -> return oly the last n real data for space {name}   
/delta/{[name0, ...]}?n                 -> return the last n real data for spaces [name0, name1, ...]  
/series/reference?x0?x1                 -> return reference data for all spaces in an interval (time is epoch time in seconds)  
/series/reference/{name}?x0?x1          -> return reference data for space {name} in an interval (time is epoch time in seconds)  
/series/reference/{[name0, ...]}?x0?x1  -> return reference data for spaces [name0, ...] in an interval (time is epoch time in seconds)  
/series/delta?x0?x1                     -> return real data for all spaces in an interval (time is epoch time in seconds)  
/series/delta/{name}?x0?x1              -> return real data for space {name} in an interval (time is epoch time in seconds)  
/series/delta/{[name0, ...]}?x0?x1      -> return real data for spaces [name0, ...] in an interval (time is epoch time in seconds)   
/presence?x0?x1                         -> return true or false if there was a person in the given interval for all spaces
/presence/{name}?x0?x1                  -> return true or false if there was a person in the given interval for space {name}   
/presence/{[name0, ...]}?x0?x1          -> return true or false if there was a person in the given interval for spaces [name0, name1, ...]  

**API:(TBD)**  
/command?cmd=x?id=y?mac=w?val=z -> execute command x on sensor y or w with data (if necessary) z when z is an array. If both y and w are specified it returns error    
/command?cmd=macid?id=y?val=z -> assigns the id y to device with mac z of the device has currently id 0xff, mac must be passed given as a sequence if hex values like 1a:62:63:ef:32:36  
/command?cmd=list -> lists all available commands  
/command?pin=xyz -> sends debug pin xyz, answer true is accepted, nothing otherwise  

these needs to be done better (if at all)  
/dbs/retrieve/samples -> retrieve sample data from .recoverysamples if dbsupdate is set   
/dbs/retrieve/presence -> retrieve sample data from .recoverypresence if dbsupdate is set   

missing in this list but needed?    
/plan/{name}  
/plan/logo  

  
NOTE: values in val are specified as x,y,n,..   
NOTE: series API supports types: sample, entry (debug mode only). Data for the current day are only available in debug mode  

**SYSTEM VARIABLES:**  
n/a  

**CONFIGURATION:**  
See gateserver.ini and configuration.ini  file for configuration example  

**COMMAND LINE OPTIONS:**  
-debug                  : enable debug mode 
-env                    : enable development mode
-db path                : set database path  
-dc path                : set disk cache path  
-user username          : set username   
-pwd password           : set password       
-eeprom                 : enables refresh of device eeprom at every connection   
-tdl int                : TCP read deadline in hours (default 24)   
-fth int                : set failure threshold in severe mode (default 3)   
-st string              : set start time, time specified as HH:MM   

**INSTALLATION**  
Executable file: gateserver(.exe)  
Configuration files: gateserver.ini, configuration.ini, measurement.ini, access.ini    
Resource folders: 

**TO BE DONE (in priority order)**  
 - API and webpage  
 - add script on new data (all new data?)  
 - clean code  


