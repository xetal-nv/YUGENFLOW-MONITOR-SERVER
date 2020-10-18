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
/info                               -> installation information  
/connected                          -> list all connected device which have not been marked invalid and report if they are active or not  
/invalid                            -> list all connected device which have  been marked invalid and report if the invalidity timestamp  
/measurements                       -> returns the definition of all active measurement  
/data                               -> return latest measurements for all spaces  
/latestdata/{name}                  -> return latest data for space {name}   
/latestdata/{[name0, name1, ...]}   -> return latest data for spaces [name0, name1, ...]  
/reference/?n                       -> return the last n reference measurements for all spaces  
/reference/{name}?n                 -> return oly the last n reference measurements for space {name}   
/reference/{[name0, ...]}?n         -> return the last n reference measurements for spaces [name0, name1, ...]  
/real/?n                            -> return the last n real data for all spaces  
/real/{name}?n                      -> return oly the last n real data for space {name}   
/real/{[name0, ...]}?n              -> return the last n real data for spaces [name0, name1, ...]  

**API:(TBD)**  
/series?type=y?space=z?analysis=y?start=x0?end=x1 -> samples of type y from space z and analysis y from timestamp x0 to timestamp x1  
/presence?space=z?analysis=y?start=x0?end=x1 -> samples of type presence from space z and analysis y from timestamp x0 to timestamp x1, the value is set to 2 when activity is detected in the period and the value is equal to the number of detection at the end of the period  
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
 - clean code  


