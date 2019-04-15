# gateserver

Copyright Xetal @ 2019  
Author: F. Pessolano  


**List API:**  
HTTPSPORTS[1]/dvl -> developer log  
HTTPSPORTS[1]/asys -> information on all current analyses  
HTTPSPORTS[1]/info -> installation information  
HTTPSPORTS[1]/active -> list all valid connected devices  
HTTPSPORTS[1]/und -> list all connected devices not used in the installation  
HTTPSPORTS[1]/udef -> list all connected devices with undefined id 0xff  
HTTPSPORTS[1]/x/y/z -> actual value for data x in space y on averaging z  
HTTPSPORTS[1]/series?last=x?type=y?space=z?analysis=y -> last x samples of type y from space z and analysis y  
HTTPSPORTS[1]/series?type=y?space=z?analysis=y?start=x0?end=x1 -> samples of type y from space z and analysis y from timestamp x0 to timestamp x1 
HTTPSPORTS[1]//cmd?cmd=x?id=y?val=z -> execute command x on sensor y with data (in necessary) z whene z is an array


**List HTTP pages:**  
HTTPSPORTS[0]/ -> simple JS JSON feed  

**SVG convention:**  
Elements triggering data from a entry need to have as ID the entry id as from the configuration file  
Elements triggering data from the full counter need to have as ID the space name  
Two classes need to be defined, st1 for unselected trigger and st2 for selected trigger  
If the server does no have a svg for a given space, the space will be ignored  

**SYSTEM VARIABLES:**  
GATESERVER is set to the application folder  

**CONFIGURATION:**  
See .env file for configuration  
