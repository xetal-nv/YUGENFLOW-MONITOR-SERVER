# gateserver

Copyright Xetal @ 2019  
Author: F. Pessolano  

**Requirements:**  
64-bit architecture  
Badger version 1.5.5  

**List API:**  
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
-env String : specifies the configuration file, uses .env if not specified  
-dbs Path : specifies path where to store the database, './DBS' used if not specified  
-dmode Int : specifies an execution mode (0: default, 1: full test, 2: short test)  
-debug Int : specifies a debug mode (0: off, -1: short flow noalgo, 1: verbose, 2: verbose no algo, 3: verbose no dbs, 4: verbose no algo no dbs)  
-dvl : activate dvl  
-ri Int : set log ri  
-rs Int64 : set log rs  
-dellogs : delete all existing logs  
-nomal : disable malicious attack control  
-norst : disable start-up device reset  
-cdelay int : specifies the maximum delay for recovery data usage  
-ks : enable killswitch API  

**CHANGELOGS FOR v0.5.0:**  

0. Overall code cleaning  
1. Server does not use latest current value in case of restart from crash, need to be depended on restart time (closed, done)  
2. Server debug needs additional modes only for the algorithm (skipped)  
3. Unclear why sometimes JS received wrong entry values (closed, bug found)  
4. Unclear why sometimes reporting hangs on /info or later calls (closed, network related)  
5. Algorithm suffers when one sensor is not present in a gate (to be tested)  
6. Tablets not supported yet, Explorer not supported, issues with chrome to be resolved (closed, user error)  
7. Add redundancy calls in JS reporting (closed, done)  
8. Entry to total, not average, and adjust interface (no sense to show non current) (done)  
9. Analysis period and start hour (in progress)  
10. Give JS analysis specs for proper reporting and simplify code  
11. Eliminate * and entry in report  (done)
12. Add API (removable at compile time) for databased analysis  
13. Entry split in and out (postponed to (0.6.0)  
14. log binary (command line) dump data per day  
15. Remove interpolation from js (done)  
16. Check sample to 0 in closure  (done, seems to work)
17. Check reset at start-up again (add CRC check)
18. Removed bug not forcing entries to zero in closure time (done)  
19. Make report on current optional  
20. Added kill switch (done)  
21. Removed rdundant mutex and added few more in exeParamCommand (done)

