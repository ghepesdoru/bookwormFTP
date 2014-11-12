#BookwormFTP
##BookwormFTP Client (github.com/ghepesdoru/bookwormFTP/client)
BookwormFTP Client is FTP client that can serve as a quick download/upload solution in your application. The client is constructed around BookwormFTP building blocks and will take care of server control and data connections, file downloads in specified downloads directory, credentials etc. 
The client will try it's best to enforce most recent available features as described by the server's FEAT response, but will automatically remove any optional feature if the server responds with a 502 (Not implemented) or 202 (Command not implemented) reply codes.

###Client Initialization
####Default client instantiation
Default client initialization is made throw the usage of <b>.NewClient("URL")</b> and will return the first encountered error or a reference to the connected client. 
While connecting the client will execute a series of commands based on the specific context:

* <b>LogIn</b>: the client will automatically authenticate using the URL encoded user and password or fallback on default anonymous login credentials (<b>USER</b>, <b>PASS</b>). If the server demands an account in the login sequence, the client will fail to complete the sequence (read the custom initialization method to learn more).  
* <b>System type query</b>: determines the system type (<b>SYST</b>)
* <b>Feature detection</b>: query the server to detect supported features (<b>FEAT</b>), and generates a Feature structure from collected data. The resulting features structure will be used to predetermine support before usage of equivalent commands (newer will always be preferred).
* <b>Determine the server side current directory</b>: executes a <b>PWD</b> command and uses the resulting path as current directory in the server PathManager
* <b>Enforces the connection parameters</b>: Representation type (<b>TYPE</b>) ASCII, format control NonPrint, Transfer mode (<b>MODE</b>) STREAM, FileStructure (<b>STRU</b>) FILE.
* <b>Changes the current directory to the URL encoded path</b>: (<b>CWD</b>)
   
By default the client instance will try to first establish a IPv4 connection, followed by another try to establishing a IPv6 connection. Only after testing both connection types, the client will fail connecting with an error.
If your specific context requires, explicitly specifying the IP version in use, two other client builders are available for this specific purpose: <b>.NewIPv4("URL")</b> and <b>.NewIPv6("URL")</b>. Neither of them initializes the client in the above specified manner.
```
package main

import (
  "fmt"
  Client "github.com/ghepesdoru/bookwormFTP/client"
)

func main() {
/* Example of client connection to the Mozzila FTP server */
  c, err := Client.NewClient("ftp.mozilla.org/pub/mozilla.org/")
  
  /* 
   * Early error management is a suggestion in golang world, 
   * but it's a must in this case, the returned *Client reference 
   * being nil in case of error 
  */
  if err != nil {
      fmt.Println("Client initialization error: ", err)
  }
  
  /* Do something with the connected client at this point */
}
```

####Custom client initialization to specify account and virtual host
If the requested connection requires an account or virtual host specified for a valid connection, a manual client initialization is required. You can use either one of the IP version specific builders, and initialize a new client, then specify the account using the client's <b>Account</b> method and a virtual host throw the usage of client's <b>Host</b> method.

The manual initialization won't execute any of the default client commands at initialization time giving the full flexibility of a totally custom connection.
To establish the current connection in a fully functional connection, at least a few of the automated default client initialization commands are required:

* <b>LogIn</b>: authenticate with specified credentials throw the usage of <b>LogIn</b> method. If the required credentials are provided in the URL, the method can be used with a nil parameter, a fallback to the parsed Credentials structure is being made. If however you wold like to provide the credentials manually, it can be done throw initialization of a local Credentials structure (<b>/core/credentials</b> package). If for any reasons the connection will have to reset using other credentials, a Login with the new Credentials structure can be used, it will take care of connection reinitialization (<b>REIN</b>). 
* <b>Feature detection:</b> the client has to know server supported feature before usage. This can be done using the <b>Features()</b> client method.
* <b>Determine the server side current directory</b>: Required step if the initial client path differs from the default root directory <b>/</b>.
```
/* Manual initialization of an IPv4 connection to the server */
c, err := Client.NewIPv4("URL")

/* Set the client account data */
c.Account("Account_information_string")

/* Set the client host (this has to be set before login in or a 
* reinitialization will be required) */
c.Host("virtualHostName")

/* Use non url embedded credentials example
* (Credentials "github.com/ghepesdoru/bookwormFTP/core/credentials") 
*/
credentials, err := Credentials.NewCredentials("username", "password")
if err != nil {
  panic("Wrong credentials.")
}

/* Login (will take account information into account if requested by server) */
c.LogIn(credentials)
 ```
###Downloading
####Fast download
If your only requirement is to download a specific file or recursively an entire directory and all it's contents, you can establish a client instance specifically for this purpose using .NewDownload("URL"). This builder reuses .NewClient and will try to establish a new connection on first available IPv.
```
/* 
* Example of a client initialization with the purpose of downloading 
* the entire publicly accessible mozzilla.org directory 
* (in reality usage of their server like this is in contradiction 
* with the server rules, so don't!) 
*/
c, err := Client.NewDownload("ftp.mozilla.org/pub/mozilla.org/")
```
####Download with an initialized client
Any client can download at any time any specified resource (file or directory) at any time. 
    
    /* Download the resource */
    ok, err = c.Download("resourceName")
     
    
###File and directory removal
At any point, using any initialized client, any resource from any path can be removed using the client's method <b>Delete</b>()
```
/* Delete the resource by it's path (being it relative or absolute) */
ok, err = c.Delete("resourceNameOrPath")
```
##Advanced usage cases
###Unmanaged commands
If you require to use any of the commands not externalized by the client, direct command querying is possible throw the usage of .Commands. Most commands will reply with a success execution flag and the eventual error in case of failure, but each command that should return a meaning full reply will do this in plain string or throw one of the core library types (for example FEAT will return a Features structure, LIST and MLSD will return a Resource structure, etc.)
```
/* Manual commands invocation examples  */
serverHelpReply, err = c.Commands.Help("with")
systemType, err = c.Commands.SYST()
ok, err = c.Commands.CWD("/desired/path/to/change/to")
```    
