package main

import (
    "apiBatchRequester/core"
    "flag"
    "fmt"
    "github.com/bytedance/sonic"
    "gopkg.in/yaml.v3"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "os"
    "strings"
    "time"
)

var (
    usage = `ApiBatchRequest_script 1.0 , usage example:
./apiBatchRequester  -f ./xx/examole1.yaml   // 根据配置文件(.json .yaml)
（detail： https://github.com/zxianx/ApiBatchRequestor/readme.md）
`
    confFile string
)

func init() {
    flag.StringVar(&confFile, "f", "", "配置文件（.json .yaml）路径")
}

func main() {
    fmt.Println(usage)
    fmt.Println("ARGS: ", os.Args)

    abrConf := parseConf()

    if abrConf.Timeout > 0 {
        core.DefaultHttpClient.Timeout = time.Duration(abrConf.Timeout) * time.Millisecond
    }
    if abrConf.Proxy != "" {
        tranport := core.DefaultHttpClient.Transport.(*http.Transport)
        proxyURL, err := url.Parse(abrConf.Proxy)
        if err != nil {
            log.Fatal("parse conf proxy err", err)
        }
        tranport.Proxy = http.ProxyURL(proxyURL)
    }

    poster, err := abrConf.NewPoster()
    if err != nil {
        log.Fatal(err)
    }

    if err = poster.Run(); err != nil {
        log.Fatal(err)
    }

}

func parseConf() (abrConf core.ApiPosterConf) {
    flag.Parse()
    if confFile != "" {
        log.Println("conf file: ", confFile)
        fContent, err := ioutil.ReadFile(confFile)
        if err != nil {
            log.Fatal("read conf file err :", err)
        }
        if strings.Contains(confFile, ".json") {
            err = sonic.Unmarshal(fContent, &abrConf)
            if err != nil {
                log.Fatal("parse conf file jsonUnmarshal  err :", err)
            }
        } else if strings.Contains(confFile, ".yaml") {
            err = yaml.Unmarshal(fContent, &abrConf)
            if err != nil {
                log.Fatal("parse conf file yamlUnmarshal  err :", err)
            }
        } else {
            log.Fatal("unSupport confFile type")
        }

    } else {
        log.Fatal("not find confile")
    }
    return
}
