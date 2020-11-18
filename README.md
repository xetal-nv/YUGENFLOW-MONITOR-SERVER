# Xetal Flow Monitoring GateServer

## 1. Specifications

Copyright Xetal @ 2020  
Author: F. Pessolano  
Version: 2.0.0 Alpha

**THIS VERSION BREAKS BACK COMPATIBILITY**

**1.1 REQUIREMENTS**  

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
 
External services:
  - mongoDB database  
  
Detachable services:  
  - webService  

**1.2 API:**  
/info                                   -> installation information  
/connected                              -> list all connected device which have not been marked invalid and report if they are active or not  
/invalid                                -> list all connected device which have been marked invalid and report the invalidity timestamp  
/measurements                           -> returns the definition of all active measurement  
/latestdata                             -> return latest measurements for all spaces  
/latestdata/{name}                      -> return latest data for space {name}   
/latestdata/{[name0, name1, ...]}       -> return latest data for spaces [name0, name1, ...]  
/reference?n                            -> return the last n reference measurements for all spaces  
/reference/{name}?n                     -> return oly the last n reference measurements for space {name}   
/reference/{[name0, ...]}?n             -> return the last n reference measurements for spaces [name0, name1, ...]  
/series/reference?x0?x1                 -> return reference data for all spaces in an interval (time is epoch time in seconds)  
/series/reference/{name}?x0?x1          -> return reference data for space {name} in an interval (time is epoch time in seconds)  
/series/reference/{[name0, ...]}?x0?x1  -> return reference data for spaces [name0, ...] in an interval (time is epoch time in seconds)   
/presence?x0?x1                         -> return true or false if there was a person in the given interval for all spaces
/presence/{name}?x0?x1                  -> return true or false if there was a person in the given interval for space {name}   
/presence/{[name0, ...]}?x0?x1          -> return true or false if there was a person in the given interval for spaces [name0, name1, ...]  
/command/{cd}?id=y?mac=w?val=z?async=0/1-> execute command cd with specified id, mac and/or data val. If async is given and set to 1, it will not wait for execution to be completed  

_To be added with webapp:_    
/plan/{name}  
/plan/logo  

_For development only:_    
/delta?n                                -> return the last n raw data points for all spaces  
/delta/{name}?n                         -> return oly the last n raw data points for space {name}   
/delta/{[name0, ...]}?n                 -> return the last n raw data points for spaces [name0, name1, ...]  
/series/delta?x0?x1                     -> return raw data for all spaces in an interval (time is epoch time in seconds)  
/series/delta/{name}?x0?x1              -> return raw data for space {name} in an interval (time is epoch time in seconds)  
/series/delta/{[name0, ...]}?x0?x1      -> return raw data for spaces [name0, ...] in an interval (time is epoch time in seconds)  

**1.3 SYSTEM VARIABLES:**  
n/a  

**1.4 CONFIGURATION:**  
The configuration uses several ini files as follows (refer to the ini itself for a detailed description):  

access.ini : it sets access to the server by configuring and enabling database, tcp connection, external scripting and alike  
configuration.ini : it sets the installation in terms of gates, entries and spaces  
gateserver.ini : it contains all settings of the server itself  
measurement.ini : it contains the definitions of all emasureemnts the server needs to precomputer and provide apart form the real time data  
webservice.ini : sets the web service (incomoplete and will be separated into a separate client server)    

**1.5 COMMAND LINE OPTIONS:**  
-db path                : set database path  
-dc path                : set disk cache path  
-debug                  : enable debug mode 
-delogs                 : delete all logs  
-eeprom                 : enables refresh of device eeprom at every connection   
-export                 : enable export mode  
-fth int                : set failure threshold in severe mode (default 3)   
-pwd password           : database password       
-tdl int                : TCP read deadline in hours (default 24)   
-st string              : set start time, time specified as HH:MM   
-user username          : database username   

_For development only:_    
-dev                    : development mode  
-echo                   : server enter in echo mode and data is not processed  


**1.6 INSTALLATION**  
Executable file: gateserver(.exe)  
Configuration files: see 1.4   
Resource folders: 

**1.7 BUILD OPTION**  
The following tags can be used for specific build:  
 - (notags)     : complete server build  
 - "embedded"   : build without database support  
 - "notest"     : build without testing support 
 
For minimum build size use also -a -gcflags=all="-l -B -wb=false" -ldflags="-w -s"  

## 2. API Description  

**3.1 INFO**  
The API return information about the installation as provided in the configuration.ini including possible run-time modifications.  
It returns a JSON ARRAY of the following format:  

    [
      {
        "spacename": "h0",
        "entries": [
          {
            "entryName": "e0",
            "gates": [
              {
                "gateName": "kit0",
                "devices": [
                  {
                    "deviceId": 0,
                    "reversed": false,
                    "suspected": false,
                    "disabled": false
                  },
                  ...
                ],
                "reversed": false
              }
            ],
            "reversed": false
          },
          ...
        ]
      },
      ...
    ]
    
It is am array of JSON elements each describing a defined space as a name and a list of entries.  

    "spacename": "h0",
    "entries": [
        ...
    ]

Each entry is described with a name, a list of gates and a flag indicating if the gate is mounted reversed. 

    "entryName": "e0",
    "gates": [
      {
        ...
    ],
    "reversed": false

Each gate us described with a name, a list of devices and a flag indicating if the gate is mounted reversed. 

    "gateName": "kit0",
    "devices": [
      ...
    ],
    "reversed": false
    
Each device is described with its ID, a flag indicating if it is reversed, a flag indicating if the device is suspected to be broken or malicious, and a flag indicating if the device has been disabled.    
                  
    {
    "deviceId": 0,
    "reversed": false,
    "suspected": false,
    "disabled": false
    },
      
**2.1 CONNECTED**  
The API return a JSON ARRAY containing data about all connected devices. Each device is described by means of its mac address (without : symbol) and a flag indicating of the sensor is active (sending data) or not.  

    [
      {
        "mac": "0a0b0c010201",
        "active": true
      },
      ...
    ]
    
**2.2 INVALID**  
The API return a JSON ARRAY containing data about all connected invalid devices. Each device is described by means of its mac address (without : symbol) and a the timestamp (unix Epoch format) when it was marked as invalid.  

    [
      {
        "mac": "0a0b0c010201",
        "timestamp": 12345678
      },
      ...
    ]
    
**2.3 MEASUREMENT**  
The API return a JSON ARRAY describing all defined measurements.  
Each measurement is given by means of its name, its type and the predicity in seconds.  

    [
      {
        "name": "ten",
        "type": "realtime",
        "interval": 10
      },
      ...
      {
        "name": "twenty",
        "type": "reference",
        "interval": 20
      }
    ]

The type can be 'realtime' when the measurement is done as a sliding window of 'period' seconds, or 'reference' and the measurement is taken periodically every 'perdiod' seconds.  
    
**2.4 LATEST**  
The API return a JSON ARRAY containing data of all latest measurement for all spaces (if no space is indicated) or for a given list of spaces.  
The JSON array is as follows:  

    [
      {
        "space": "h0",
        "type": "reference",
        "measurements": {
          'name_measurement': [data],
          ...
        }
      },
      ...
    ]

The field space is the space name and the type can be of value 'realtime' or 'reference' as defined in section 2.3.  
The results are given in the 'measurements' list, where each element is named according to the measurement name and
it is associated to an array of measurement data. This data is expressed as follows:  

    "twenty": [
    {
      "qualifier": "twenty",
      "space": "h0",
      "timestamp": 1605712842,
      "value": 0,
      "flows": {
            ...
      }
    },
    ]
    
In this example the measurement name is 'twenty' referring to a measurement of this name as provided in the measurement.ini file. 
In a real situation thje name will depend on what the user has provided at installation time.   
The array of data contains for this API only one value (the latest one), where 'qualifier' is the measurement name (it can be also empty), 'space' is the space name (it can be also empty),
'timestamp' is the timestamp expressed as Unix Epoch time and 'value' is the measurement value.  
Furthermore, the measurement provides the a list with all flows reported by each entry belonging to the space.  
Each element of this list is as follows:  

    "e1": {
      "id": "e1",
      "Ts": 1605712842503580600,
      "netflow": 1,
      "reversed": false,
      "flows": {
        "kit1": {
          "id": "kit1",
          "variation": 1
        }
      }
    }
    
It has a field of name equal to the entry name (as form configuration.ini) and as value a JSON with field 'id' repeting the entry name, 'Ts' the measurement timestamp in Unix Epoch format, 
'netflow' that is the current difference in in- and outflow measured by the entry and 'flows' that provdes a list of flows from all gates composing the entry.  
Per gate a field is included in such list of name equal to the gate name (as form configuration.ini) and values 'id' (the gate name) and 'variation' (the different between
in- and out-flow measured by the gate).   

**2.5 REFERENCE**  
The API reference returns  a number of results for one or more spaces according to how it is called.  
It answers with a JSON identicaly to the one described in section 2.4 except that the 'type' field is always equal to 'reference' and multiple measurement data are given in the data array:  

    [
      {
        "space": "h0",
        "type": "reference",
        "measurements": {
          "twenty": [
            ...,
            ...
          ],
          ...
        }
      },
      ...
    ]

**2.6 SERIES/REFERENCE**  
Like for the 'REFERENCE' API this API return several measurement data and specifically the data for one or more spaces in the given time interval with reference times expressed in Epoch Unix format.  
The JSON is identical as the one described in section 2.5.  

**2.7 PRESENCE**  
This API is used to check if thgere was somebody in a given space in the given time internal. Start and end times must be expressed in Epoch Unix format.  
The answer is a JSON array including data for one or more spaces and expressed as follows:  

    [
      {
        "space": "h0",
        "presence": false
      },
      ...
    ]

Where 'space' is the space name and 'presence' is true of there was a person in the space in the given interval, otherwise it is false.  

**2.8 COMMAND**  

## 3. External scripting    

## 4. Release Notes  

**3.1 Known bugs**  
This build is currently in alpha, therefore several bugs are still present  

**3.2 Feature Roadmap**  
 - Check API under nodatabase built  
 - Remove webservice
 - Verify the API for reading latest values goves latest and not first values  
 - Check export manager as it seems that fields have \r\n characters in them  
 - Add calculated flows to export data and json ?  
 - Turn all non commands into GET and adjust the way request data is retrieved (r.URL.Query())  
 - Move WebService out  
 - Add database management tools  
 - Code Cleaning    


