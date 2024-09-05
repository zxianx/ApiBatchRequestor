# ApiBatchRequestor (abr) - Single-machine API Batch Request Tool

(translate by gpt3.5)

```
./apiBatchRequester  -f  ./demo.yaml   
# abr, start!!!
# Run in background: nohup ./apiBatchRequester  -f  ./demo.yaml   >  demo.log 2>&1 &
# Configuration files support JSON and YAML formats. apiBatchRequester has no environment dependencies and can be directly downloaded and used as an executable file.
```

## Introduction

​	abr is a command-line tool for batch constructing request parameters and making HTTP GET and POST requests, supporting JSON parameter formatting. It includes functionalities such as data source file parsing, parameter construction, batch requesting, rate limiting, concurrency control, time consumption statistics, error checking and logging, result parsing,
and storage.
​	The core logic of abr is to parse configuration, construct request parameters line by line based on the configuration,
and make HTTP requests. It can achieve over 10,000 queries per second on a 4-core low-pressure CPU Mac.
​	You can use abr for tasks such as batch data processing, API cleansing, and stress testing.

## Quick Start

### Example 1: Constructing POST Requests Based on a CSV Table

Target POST API and Response Example:

```shell
POST  https://abr.test.com/userInfo/setInfo
Parameter Example
{"id":123,"name":"张三"}
Response Example
{
  "errNo":0,
  "msg":"succ"
  "data":{}
}

```

Data Source File: idName.csv

```
id,name 
123,张三
124,张四
125,张五
```

Configuration File getUserInfo.yaml

```yaml
# 基本参数
# Basic Parameters
method: POST
host: https://abr.test.com
path: /userInfo/setInfo
# Parameter Construction Syntax
postParamTemplateV2: '{"id":"$1", "name":$2, "otherParam":"abc"}'  # Detailed construction syntax see below, here $1 represents the first column of each line in the file
# Source File
srcFilePath: ./idName.csv     # Data source file
# Concurrency Control
workerCoroutineNum: 50       # Maximum number of concurrent requests
qpsLimit: 100                # Rate limit, maximum number of requests per second
# Error Checking
resErrNoName: errNo          # Error probe, triggers an error if the field in the interface result is non-zero
# Result Recording configuration see below

```

Running abr will construct and send 3 httpPost requests

```
POST https://abr.test.com/userInfo/setInfo  {"id":"123", "name":"张三", "otherParam":"abc"}
POST https://abr.test.com/userInfo/setInfo  {"id":"124", "name":"张四", "otherParam":"abc"}
POST https://abr.test.com/userInfo/setInfo  {"id":"125", "name":"张五", "otherParam":"abc"}

```

### Example 2: Making GET Requests Based on an ID List File and Saving Result Data

Target GET API and Response Example:

```shell
GET  https://abr.test.com/userInfo/getById?id=123

Response Example
{
  "errNo":0,
  "data":{
    "id":123,
    "name":"张三"
  }
}

```

ID List File: idList.txt

```
123
124
125
```

Configuration File getUserInfo.yaml

```yaml
# Basic Parameters
method: GET
host: https://abr.test.com
path: /userInfo/getById
# Parameter Construction Syntax
getParamTemplateV2: '?id=$0'  # Detailed construction syntax see below, here $0 represents the entire line of each file
# Source File
srcFilePath: ./idList.txt     # Data source file
# Concurrency Control
workerCoroutineNum: 50       # Maximum number of concurrent requests
qpsLimit: 100                # Rate limit, maximum number of requests per second
# Error Checking
resErrNoName: errNo          # Error probe, triggers an error if the field in the interface result is non-zero
# Result Parsing
saveResDataExtractor: "data"  # gjson syntax, extracts the 'data' field from the result, detailed result parsing and storage configuration see below

```

After running abr, it will send 3 httpGet requests

```
GET https://abr.test.com/userInfo/getById?id=123
GET https://abr.test.com/userInfo/getById?id=124
GET https://abr.test.com/userInfo/getById?id=125

```

Generate a file named ./idList.txt_res with the following content

```
{"id":123,"name":"张三"}
{"id":124,"name":"张四"}
{"id":125,"name":"张五"}
```

## Configuration/Configuration Item List

Configuration files support JSON and YAML formats, and all configuration items are single-layer structures.
Below are the corresponding Go structures for configuration items, with JSON and YAML configuration names given in the
tags, where items with asterisks are commonly used configuration items.

```go
//1 Target interface static configuration, helping abr determine a request template
//*1.1 Service address of the request
Method string `json:"method" yaml:"method"` // GET POST
Host   string `json:"host" yaml:"host"`     //  http://domain 、 http://ip:port 
Path   string `json:"path" yaml:"path"` //  Must start with a slash
//1.2 Header information, optional
Cookies map[string]string `json:"cookies" yaml:"cookies"`
ReqHost string            `json:"reqHost" yaml:"reqHost"`
Header  map[string]string `json:"header" yaml:"header"`


//2 Source file information, letting abr know how to parse the source file
//*2.1 Source file path
SrcFilePath  string `json:"srcFilePath" yaml:"srcFilePath"`
//2.2 Only process part of the lines in the source file, similar to skip and limit in SQL
SrcFileSkip  int    `json:"srcFileSkip" yaml:"srcFileSkip"` // Default 0, no skipping
// (tips When processing CSV files, you can set it to 1 to skip the header row)
SrcFileLimit int    `json:"srcFileLimit" yaml:"srcFileLimit"` // Default 0, no limit
// (tips, you can set it to 2 to test running 2 lines)
//*2.3 Source file row-column structure, used with parameter construction template (tips Of course, the source file can also be each line with one complete parameter without construction)
SrcFileColumNum       int    `json:"srcFileColumNum" yaml:"srcFileColumNum"` // Default 1
SrcFileColumSeparator string `json:"srcFileColumSeparator" yaml:"srcFileColumSeparator"` // Default comma ","
//2.4 Special source file format, usually not manually specified
SrcFileSepUsRsUs  bool   `json:"srcFileSepUsRsUs" yaml:"srcFileSepUsRsUs"` // Use "\x1E" as row separator and "\x1F" as column separator, (RS, US are used as column separators in files, commonly used to avoid conflicts between separators and table cell content)
SrcFileLineTrim   string `json:"srcFileLineTrim" yaml:"srcFileLineTrim"`   // Default "\n\r\t"
//2.5 Row merging, multiple lines to 1 line and multiple columns (does not affect rate limiting, still counts as multiple lines for rate limiting)
MultiLine          int `json:"multiLine" yaml:"multiLine"`                   // Default 1, i.e., no merging
MultiLineJoinStr   string  `json:"multiLineJoinStr" yaml:"multiLineJoinStr"` // Default ","
// (tips: For example, if you have a single-column ID file, you can set multiLine=3 to merge 3 lines into 1 line with 3 columns, and the merged single line is "id1,id2,di3", suitable for parameter construction in scenarios like mget)

//3 Request parameter construction, giving abr a parameter construction template
// (tips: Currently, post requests only support constructing JSON parameters, and do not support constructing query and body simultaneously)  
//*3.1 Method 1, not constructing, each line in the file is a request parameter, the file line format requirements are as follows
// get request format: ?aa=1&bb=xx ,
// post request JSON format: {"a":1,"b":"bv"}
ParamDirect bool `json:"paramDirect" yaml:"paramDirect"`
//*3.2 GET request string template parameter construction
// Parameter format eg ?aa=$1&bb=$2 (replace $n with the nth column in the file line, $0 is to replace the entire line)   
GetParamTemplateV2 string `json:"getParamTemplateV2" yaml:"getParamTemplateV2"`
GetUsePathTemplate bool `json:"getUsePathTemplate" yaml:"getUsePathTemplate"`
// If GetParamTemplateV2 is not present and GetUsePathTemplate == true, the path will be treated as a parameterized template.
//*3.3 Post string template construction
PostParamTemplateV2 string `json:"postParamTemplateV2" yaml:"postParamTemplateV2"`
// eg {"a":$1,"a2":xxx_$1,"b":"$2","c":$.JSON3}  (replace the nth column in the file line with $n, $0 is the entire line, $.JSON3 indicates that $3 is replaced after JSON encoding)
//(tips: Usually use $n for numbers, "$n" for simple strings, $.JSONn for strings that need to be escaped (parentheses will be automatically added)
// 3.4 Advanced Parameter Construction
// See the following text for embedded custom function construction, general parameter supplementation construction, JavaScript functions, executable file construction, and other constructions for request parameters.

//*4 Request rate limiting, concurrency limiting
QpsLimit   int `json:"qpsLimit" yaml:"qpsLimit"`                // Maximum rate
QPerTimeRange   int `json:"qPerTimeRange" yaml:"qPerTimeRange"` // Rate limiting interval, in seconds, default 1s, i.e., n q/1s, applicable to rate limiting < 1qps
WorkerCoroutineNum  int `json:"workerCoroutineNum" yaml:"workerCoroutineNum"`
//Concurrency, default is 1 for serial requests, and the maximum number of requests to be sent to the service at the same time
ExpectReqCostMillisecond int `json:"expectReqCostMillisecond" yaml:"expectReqCostMillisecond"`
//Expected time consumption for a single request (milliseconds), when concurrency is not set, the system automatically arranges concurrency based on this field

//5 Error checking, result parsing, letting abr know how to detect abnormal requests and save the results you want
// By default, non-zero error codes in HTTP connection errors and non-200 status codes in HTTP responses are considered errors,
// 5.1 Discard result without checking, suitable for stress testing to improve performance of the sending end (still checks HTTP errors)
DiscardResBody bool `json:"discardResBody" yaml:"discardResBody"`
//* 5.2 Interface return result error code probe (error code field name)
ResErrNoName   string `json:"resErrNoName" yaml:"resErrNoName"`
// If there is this field, it will additionally parse the error code in the returned result. Non-zero is an error, eg errno, errNo
//* 5.3 Result parsing, storage
SaveRes              bool   `json:"saveRes" yaml:"saveRes"` // Save the return value, default is not saved
SaveResDataExtractor string `json:"saveResDataExtractor" yaml:"saveResDataExtractor"` // Default empty, i.e., save the complete returned content
// gjson path format, parse and save part of the data in the returned value as the result 
// eg If a certain interface returns a result: {"errNo":0,"data":{"a":1}} 
// saveResDataExtractor is set to data to extract '{"a":1}' ; set to data.a to extract 1
SaveResDataTemplate   string   `json:"saveResDataTemplate" yaml:"saveResDataTemplate"` // Result storage construction template,
// saveResDataTemplate is empty by default, i.e., the result parsed by SaveResDataExtractor,
// supports the original file structure $n and the parsing result of saveResDataExtractor $resDataExtract
// eg, if there are two columns in the source file, and you need to use the second column to request a certain interface, and then combine the first column and the interface output into two columns to save, then you can set it as "$1,$resDataExtract"
// SaveResDataExtractors []string `json:"saveResDataExtractors" yaml:"saveResDataExtractors"` // Output as row-column structure  Not implemented Not commonly used todo       

//6 Result, error file storage location
ResFilePath string `json:"resFilePath" yaml:"resFilePath"` // Result file location Default is srcfile path+".res"
ErrFilePath string `json:"errFilePath" yaml:"errFilePath"` // Error file location   Default is srcfile path+".err"
// Note that rows with unexpected file data, such as non-zero error codes for failed requests, will be automatically written to the err file, and the error message itself will be output to standard output

//7 Other parameters
Name string    `json:"name" yaml:"name"` // Task name, optional
DetailLog bool `json:"detailLog" yaml:"detailLog"` // Output detailed request parameters and results for each request, suitable for testing or debugging
Statistic bool `json:"statistic" yaml:"statistic"` // Continuous output of statistical information, such as interface consumption time, success rate, etc., suitable for stress testing.
Timeout int    `json:"timeout" yaml:"timeout"`     // Request timeout, in milliseconds, generally not set
Retry  int     `json:"retry" yaml:"retry"`   // Number of retries for HTTP request failure, default is 0, no retry 
DryRun  bool   `json:"dryRun" yaml:"dryRun"` // Only construct and print request parameters without actual requests
Proxy   string            `json:"proxy" yaml:"proxy"` // Proxy URL


```

## Other Tips

1. Testing

You can set the following parameters

```
## Only run the first 10 lines of the file, and print detailed request parameters and return results for each request
detailLog:true
srcFileLimit:10
dryRun:true  # Does not actually send requests

```

2. Handling unexpected interruptions in tasks
   There is currently no mechanism to continue interrupting. Generally, after the task is interrupted and restarted, you
   need to skip the executed lines by setting srcFileSkip based on the logs (not strict, it is recommended to use the
   last offset minus the rate limit qps),

## Advanced Parameter Construction and Result Parsing

### User-Defined Hook Functions

abr also provides the following hook usage, users can add custom hooks to complete complex parameter construction and
result parsing. (Need to insert code in the specified location of the project and recompile)

+ **Parameter Construction Hook**

  Hook insertion location: {project directory}/hooks/paramBuilder/

```go

// ParamBuilder  Request parameter construction hook
//  line is one line in the source file (not necessarily one line, if you specify a record separator that is not \n)
//  param is the constructed parameter returned,
//  param supports any struct type,
//  For get requests, param should be of type string, format example eg "?a=123&b=xxx"
//  For Post requests, param supports four types: any struct/map, json string/[]byte
///*
type ParamBuilder func (line string) (param interface{}, err error)

var BuiltInParamBuilderNameMap = map[string]ParamBuilder{"": nil}

func init() {
// insert your hooks here
BuiltInParamBuilderNameMap["demo"] = func (line string) (param interface{}, err error) {
param = map[string]interface{}{
"a": line,
}
return
}

}

```

+ Parameter Construction Supplementary Hooks 

Suitable scenario: In the same business, different interfaces may include the same additional parameters (such as authentication parameters, timestamps, etc.), and it is usually difficult to construct them using string templates.
A parameter construction supplementary hook can be used to complete the construction of these common or complex additional parameters, and then merge them into the parameters constructed through string templates.

Hook insertion position: {project directory}/hooks/paramAppender/

```go
// ParamAppender is a parameter construction supplementary hook that supports parameter supplementation for GET and POST requests.
// line represents a line in the source file (it may not be a single line, if you specify a record delimiter other than \n).
// append represents the additional constructed parameters to be returned. If it is a GET request, it is concatenated to the original parameters; if it is a POST request, it is merged into the outermost layer of the request parameters.
// Fields in append do not support arrays or nested structures.
type ParamAppender func(line string) (param map[string]interface{}, err error)

// BuiltInParamAppenderNameMap stores the built-in param appenders with their names as keys.
var BuiltInParamAppenderNameMap = map[string]ParamAppender{"": nil}

func init() {
    // insert your hooks here

    // Append a timestamp parameter named "time" to the parameters
    BuiltInParamAppenderNameMap["time"] = func(line string) (append map[string]interface{}, err error) {
        append = map[string]interface{}{
            "time": time.Now().Unix(),
        }
        return
    }

}

```

+ Result Parsing Hook (todo)

### External Parameter Constructor (todo)

How to achieve complex parameter construction without changing the code and recompiling?

External executable script? lua? js function? (Golang can execute JavaScript functions through Chrome v8)

## Welcome to Improve abr Together

The core logic of abr is a production-consumption model (code located at ./core/abr), with only a few hundred lines of
code and no complex structures. You are welcome to directly add the features you need and submit a pull request.
For performance optimization or more detailed HTTP request tracing, you can refer to the library https://github.com/rakyll/hey (todo).