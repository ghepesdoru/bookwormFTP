#BookwormFTP
##BookwormFTP Client (github.com/ghepesdoru/bookwormFTP/client)
BookwormFTP Client is FTP client that can serve as a quick download/upload solution in your application. The client is constructed around BookwormFTP building blocks and will take care of server control and data connections, file downloads in specified downloads directory, credentials etc. 
The client will try it's best to enforce most recent available features as described by the server's FEAT response, but will automatically remove any optional feature if the server responds with a 502 (Not implemented) or 202 (Command not implemented) reply codes.

###Client Initialization
Default client initialization is made throw the usage of .NewClient("URL") and will return the first encountered error or a reference to the connected client.
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

By default the client instance will try to first establish a IPv4 connection, followed by another try of establishing a IPv6 connection. Only after testing both connection types, the client will fail connecting with an error.
If your specific context requires, explicitly specifying the IPv to be used is possible, two other client builders being available .NewIPv4("URL") and .NewIPv6("URL").

###Downloading
If your only requirement is to download a specific file or recursively an entire directory and all it's contents, you can establish a client instance specifically for this purpose using .NewDownload("URL"). This builder reuses .NewClient and will try to establish a new connection on first available IPv.
    
    /* 
     * Example of a client initialization with the purpose of downloading 
     * the entire publicly accessible mozzilla.org directory 
     * (in reality usage of their server like this is in contradiction 
     * with the server rules, so don't!) 
     */
    c, err := Client.NewDownload("ftp.mozilla.org/pub/mozilla.org/")
    
###Realm and account settings
If your specific context requires setting a realm or an account before connecting, a manual initialization throw one of the lower lever builders is required.

    /*
     * Manual initialization of an IPv4 connection to the server
     */
    c, err := Client.NewIPv4("URL")
    
    /* Set the client account data */
    c.Account("Account_information_string")
    
    /* Set the client realm (this has to be set before login in or a 
     * reinitialization will be required) */
    
    /* Use non url embedded credentials example
     * (Credentials "github.com/ghepesdoru/bookwormFTP/core/credentials") 
     */
    credentials, err := Credentials.NewCredentials("username", "password")
    if err != nil {
        panic("Wrong credentials.")
    }
    
    /* Login (will take account information into account if requested by server) */
    c.LogIn(credentials)