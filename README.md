# Xetal Flow Monitoring GateServer

Copyright Xetal @ 2019  
Author: F. Pessolano  

**THIS VERSION BREAKS BACK COMPATIBILITY**

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
  
Detachable services:  
  - webService  

**API:** 
/info                                   -> installation information  
/connected                              -> list all connected device which have not been marked invalid and report if they are active or not  
/invalid                                -> list all connected device which have  been marked invalid and report if the invalidity timestamp  
/measurements                           -> returns the definition of all active measurement  
/latestdata                             -> return latest measurements for all spaces  
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
/command/{cd}?id=y?mac=w?val=z?async=0/1-> execute command cd with specified id, mac and/or data val. If async is given and set to 1, it will not wait for execution to be completed  

To be added with webapp    
/plan/{name}  
/plan/logo  

**SYSTEM VARIABLES:**  
n/a  

**CONFIGURATION:**  
See *.ini files for details  

**COMMAND LINE OPTIONS:**  
-db path                : set database path  
-dc path                : set disk cache path  
-debug                  : enable debug mode 
-delogs                 : delete all logs
-dev                    : development mode
-echo                   : server enter in echo mode and data is not processed  
-eeprom                 : enables refresh of device eeprom at every connection   
-export                 : enable export mode
-fth int                : set failure threshold in severe mode (default 3)   
-pwd password           : set database password       
-tdl int                : TCP read deadline in hours (default 24)   
-st string              : set start time, time specified as HH:MM   
-user username          : set database username   

**INSTALLATION**  
Executable file: gateserver(.exe)  
Configuration files: gateserver.ini, configuration.ini, measurement.ini, access.ini    
Resource folders: 

**BUILD OPTION**  
The following tags can be used for specific build:  
 - (notags)     : complete server build  
 - "embedded"   : build without database support  
 - "notest"     : build without testing support 
 
For minimum build size use also -a -gcflags=all="-l -B -wb=false" -ldflags="-w -s"  

**TO BE DONE (in priority order)**  
 - Check the API for reading latest values !!  
 - Check export manager as it seems that fields have \r\n characters in them  
 - Turn all non commands into GET and adjust the way request data is retrieved (r.URL.Query())  
 - Move WebService out  
 - Add database management tools  
 - Code Cleaning    


