# Text Files Service Mini Project

A simple RESTful-like web service in Golang that allows operating over text files as resources.

- GET: Retrieve the contents of a text file under the given path
- POST: Create a text file with content under the given path
- PUT: Replace the contents of a text file under the given path
- DELETE: Delete the resource under given given path

Also, provides a method to get statistics per folder basis. These statistics are:
- The total number of files in that folder.
- The average number of alphanumeric characters per text file (and standard deviation) in that folder.
- The average word length (and standard deviation) in that folder. 
- The total number of bytes stored in that folder.

## How to Use

Get the source code:
```
go get github.com/twsiyuan/text-files-service-mini-project
```

Modify arguments in ```main.go``` if needed
- ```port```: Listening port
- ```fileDir```: root folder that holds text files

Build Go project in the folder via the command:
```
go build .
```

Run service:
```
./text-files-service-mini-project
```

## API Examples

### Retrieve File

Request:
```
GET /news HTTP/1.1
Host: 127.0.0.1:8080
```

Response:
```
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Content-Length: 332

{"Content":"News\n\nNews is information about current events. This may be provided through many different media: word of mouth, printing, postal systems, broadcasting, electronic communication, and also on the testimony of observers and witnesses to events. It is also used as a platform to manufacture opinion for the population."}
```

### Retrieve Statistics of Folder

Request:
```
GET /news/ HTTP/1.1
Host: 127.0.0.1:8080
```

Response:
```
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Content-Length: 180

{
   "NumFiles":2,
   "AvgNumAlphaCharsPerFile":351.5,
   "StdNumAlphaCharsPerFile":60.5,
   "AvgWordLength":4.950704225352113,"StdWordLength":2.2652533508425217,
   "TotalBytes":864
}
```

### Create File

Request:
```
POST /news/today-news HTTP/1.1
Host: 127.0.0.1:8080
Content-Type: application/json; charset=utf-8
Content-Length: 26

{"Content":"Hello World!"}
```

Response:
```
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Content-Length: 6

"DONE"
```

### Replace File Example

Request:
```
PUT /news/today-news HTTP/1.1
Host: 127.0.0.1:8080
Content-Type: application/json; charset=utf-8
Content-Length: 26

{"Content":"Hello World!"}
```

Response:
```
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Content-Length: 6

"DONE"
```

### Delete File

Request:
```
DELETE /news/today-news HTTP/1.1
Host: 127.0.0.1:8080
```

Response:
```
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Content-Length: 6

"DONE"
```