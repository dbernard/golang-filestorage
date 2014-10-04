#Simple Cloud File Storage with Go

*This code needs to be cleaned up! More coming soon.*

This is a simple example of file storage in the cloud using Go. This example in
particular stores JSON files and uses simple HTTP basic authentication.

Because this is an *example*, the authentication credentials aren't really a
secret and can be setup in the source code... so here are the defaults:

username = user1
password = pass1

**Keep in mindy you NEED to provide your own database url to the
```sql.Open()``` statement.**

This app uses:
- [Golang](http://golang.org/doc/)
- [postgres SQL driver for Go](http://github.com/lib/pq)
- [Heroku | Cloud Application Platform](http://heroku.com)


See links above for setup help and documentation details.

###Local usage

To use this application locally, first compile the go app,
```
go build home.go
```

Next, run the compiled application with an environment variable "PORT" set to
the port you wish to deploy your instance on, for example,
```
PORT=8080 ./home
```

Finally, from your browser visit
```
http://localhost:8080/
```

###cURL usage

To access a running instance of this app via cURL, use the following commands:

- To upload a file:
```
curl --form username=user1 --form password=pass1 --form myfiles=@(JSON-FILE-HERE) --form press=submit (URL)
```

- To Download a file:
```
curl (URL)/download/(YOUR-FILENAME-HERE)
```

###Deploy to a Heroku server

To deploy this app to a heroku server, follow the instructions at the following
link:

[Setting up Heroku with Go](http://mmcgrana.github.io/2012/09/getting-started-with-go-on-heroku.html)

