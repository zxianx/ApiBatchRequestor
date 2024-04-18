package core

import (
    "apiBatchRequester/hooks/paramAppender"
    "apiBatchRequester/hooks/paramBuilder"
    "bufio"
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "github.com/bytedance/sonic"
    "github.com/tidwall/gjson"
    "io"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "reflect"
    "strings"
    "sync"
    "sync/atomic"
    "time"
)

const (
    MaxCoroutineNum      = 3000
    LineSeparatorDefault = "\n"
    LineSeparatorRs      = "\x1E"
    ColSeparatorUs       = "\x1F"
)

type statisticData struct {
    reqTotal      uint64
    reqFail       uint64
    statusCodeErr uint64
    parseBodyErr  uint64
    succ          uint64
    Time          uint64
}

type apiPoster struct {
    ApiPosterConf

    statisticData

    isGetMethod bool
    url         string
    cookieStr   string
    startTime   time.Time
    done        bool
    wg          sync.WaitGroup
    ch          chan string //src item ch
    errItemCh   chan string
    resCh       chan string

    checkInterval int64
}

func (c *ApiPosterConf) NewPoster() (poster *apiPoster, err error) {
    poster = &apiPoster{
        ApiPosterConf: *c,
    }

    if c.DryRun {
        poster.DetailLog = true
    }

    if c.BuiltInParamBuilderName != "" {
        poster.BuiltInParamBuilder = paramBuilder.BuiltInParamBuilderNameMap[c.BuiltInParamBuilderName]
        if poster.BuiltInParamBuilder == nil {
            err = fmt.Errorf("unknown BuiltInParamBuilderName:%s", c.BuiltInParamBuilderName)
        }
    }

    if c.BuiltInParamAppenderName != "" {
        poster.BuiltInParamAppender = paramAppender.BuiltInParamAppenderNameMap[c.BuiltInParamAppenderName]
        if poster.BuiltInParamAppender == nil {
            err = fmt.Errorf("unknown BuiltInParamAppenderName:%s", c.BuiltInParamAppenderName)
        }
    }

    poster.srcFileLineSep = LineSeparatorDefault[0]
    if c.SrcFileSepUsRsUs {
        poster.srcFileLineSep = LineSeparatorRs[0]
        poster.SrcFileColumSeparator = ColSeparatorUs
    }

    if c.MultiLineJoinStr == "" {
        poster.MultiLineJoinStr = ","
    }

    if c.SrcFileLineTrim == "" {
        poster.SrcFileLineTrim = "\n\r\t"
    }

    poster.Method = strings.ToUpper(poster.Method)
    if poster.Method == "POST" {

    } else if poster.Method == "GET" {
        poster.isGetMethod = true
    } else {
        err = errors.New("empty req method")
        return
    }

    if poster.Host != "" && poster.Path != "" {
        protocol := "http://"
        if strings.Contains(poster.Host, "http") {
            protocol = ""
        }
        poster.url = fmt.Sprintf("%s%s%s", protocol, poster.Host, poster.Path)
        if poster.isGetMethod {
            /*            if poster.SaveRes == false {
                          if !poster.DiscardResBody {
                              poster.SaveRes = true
                              fmt.Println("INFO： GET method auto set saveRes")
                          }
                      }*/

            if c.GetParamTemplate+c.GetParamTemplateV2 == "" && !c.ParamDirect {
                err = errors.New("GET method need GetParamTemplate, GetParamTemplateV2,or set ParamDirect")
                return
            }
            if c.GetParamTemplate != "" && c.GetParamTemplateV2 != "" {
                err = errors.New("GET method both set  GetParamTemplate, GetParamTemplateV2")
                return
            }
            poster.url += c.GetParamTemplate + c.GetParamTemplateV2
        }
    } else {
        err = errors.New("illegal host path")
        return
    }
    log.Println("url: ", poster.url)

    if poster.SrcFileColumNum == 0 {
        poster.SrcFileColumNum = 1
    }

    if poster.QpsLimit == 0 {
        err = errors.New("need QpsLimit")
        return
    }

    if poster.WorkerCoroutineNum != 0 {

    } else if poster.ExpectReqCostMillisecond != 0 {
        poster.WorkerCoroutineNum = poster.QpsLimit * poster.ExpectReqCostMillisecond / 1000 * 2 // *2 保守
    } else {
        poster.WorkerCoroutineNum = 1
    }

    if poster.WorkerCoroutineNum > MaxCoroutineNum {
        log.Println("WARN , max WorkerCoroutineNum limit, set to ", MaxCoroutineNum)
        poster.WorkerCoroutineNum = MaxCoroutineNum
    }
    log.Println("WorkerCoroutineNum", c.WorkerCoroutineNum)

    if poster.SrcFilePath == "" {
        err = errors.New("need SrcFilePath")
        return
    }

    if poster.SrcFileLimit == 0 {
        poster.SrcFileLimit = 1000000000
    }

    if poster.ResFilePath == "" {
        poster.ResFilePath = poster.SrcFilePath + ".res"
    }
    if poster.ErrFilePath == "" {
        poster.ErrFilePath = poster.SrcFilePath + ".err"
    }

    if poster.QPerTimeRange == 0 {
        poster.QPerTimeRange = 1
    }

    poster.checkInterval = int64(poster.QPerTimeRange) * int64(time.Second)

    poster.startTime = time.Now()
    poster.ch = make(chan string, poster.WorkerCoroutineNum*2)
    poster.errItemCh = make(chan string, poster.WorkerCoroutineNum)
    if poster.SaveResDataExtractor != "" || poster.SaveResDataTemplate != "" {
        poster.SaveRes = true
    }
    if poster.SaveRes {
        poster.resCh = make(chan string, poster.WorkerCoroutineNum)
    }
    if poster.Cookies != nil {
        cook := ""
        for k, v := range poster.Cookies {
            cook = cook + k + "=" + v + ";"
        }
        poster.cookieStr = cook[0 : len(cook)-1]
    }

    fmt.Println("****checked conf:\ts****")
    res, _ := sonic.MarshalString(poster.ApiPosterConf)
    fmt.Println(res)
    fmt.Println("****checked conf:\te****")
    return
}

func (c *apiPoster) itemProducerRun() (err error) {
    file1, err := os.Open(c.SrcFilePath)
    if err != nil {
        log.Println(err)
        return
    }
    defer file1.Close()
    rd1 := bufio.NewReader(file1)

    for i := 0; i < c.SrcFileSkip; i++ {
        if _, err = rd1.ReadString(c.srcFileLineSep); err != nil {
            log.Println(err)
            return
        }
    }
    fNoEOF := true
    readedInOneSecond := 0
    perSecondBegin := time.Now().UnixNano()
    multiLineJoin := ""
    for j := 0; j < c.SrcFileLimit && fNoEOF; j++ {
        line, err := rd1.ReadString(c.srcFileLineSep)
        if err != nil {
            if err == io.EOF {
                fNoEOF = false
            } else {
                log.Println(err)
                return err
            }
        }
        line = strings.Trim(line, c.SrcFileLineTrim)
        if line == "" {
            continue
        }
        readedInOneSecond++
        if c.MultiLine <= 1 {
            c.ch <- line
        } else {
            if j%c.MultiLine == 0 {
                c.ch <- multiLineJoin
                multiLineJoin = line
            } else {
                multiLineJoin += c.MultiLineJoinStr + line
            }
        }
        if c.QpsLimit > 0 && readedInOneSecond >= c.QpsLimit {
            now := time.Now()
            time.Sleep(time.Duration(c.checkInterval - (now.UnixNano() - perSecondBegin)))
            fmt.Printf("no. %d line[%s], %v \n", j+c.SrcFileSkip+1, line, now)
            perSecondBegin = perSecondBegin + c.checkInterval
            readedInOneSecond = 0
        }
    }
    if multiLineJoin != "" {
        c.ch <- multiLineJoin
    }
    return
}

func (c *apiPoster) errItemSaverRun() (err error) {
    failfile, err := os.OpenFile(c.ErrFilePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
    if err != nil {
        log.Println("errItemSaverRun OpenFile err", err, c.ErrFilePath)
        return
    }
    defer failfile.Close()
    fWr := io.Writer(failfile)
    bfwL := bufio.NewWriter(fWr)
    for item := range c.errItemCh {
        bfwL.WriteString(item + "\n")
    }
    err = bfwL.Flush()
    if err != nil {
        log.Println(err)
        return err
    }
    return
}

func (c *apiPoster) resItemSaverRun() (err error) {
    failfile, err := os.OpenFile(c.ResFilePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
    if err != nil {
        log.Println(err)
        return
    }
    defer failfile.Close()
    fWr := io.Writer(failfile)
    bfwL := bufio.NewWriter(fWr)
    for item := range c.resCh {
        _, err = bfwL.WriteString(item + "\n")
        if err != nil {
            log.Println("resItemSaverRunBfWriteStringErr", err)
        }
    }
    err = bfwL.Flush()
    if err != nil {
        log.Println(err)
        return err
    }
    return
}

func (c *apiPoster) itemPost(item string) (err error, resData string) {
    var param interface{}
    if c.ParamDirect {
        if item == "" {
            return
        }
        param = item
    } else if c.PostParamTemplate != "" {
        paramStr := c.PostParamTemplate
        if c.SrcFileColumNum == 1 {
            paramStr = strings.Replace(paramStr, "%s", item, -1)
        } else {
            arr := strings.Split(item, c.SrcFileColumSeparator)
            if len(arr) != c.SrcFileColumNum {
                err = errors.New("illegal srcFile line")
                return
            }
            for i := 0; i < c.SrcFileColumNum; i++ {
                paramStr = strings.Replace(paramStr, "%s", arr[i], 1)
            }
        }
        param = paramStr
    } else if c.PostParamTemplateV2 != "" {
        param, err = TemplateReplace(c.PostParamTemplateV2, item, c.SrcFileColumSeparator)
        if err != nil {
            return err, ""
        }
    } else if c.BuiltInParamBuilder != nil {
        param, err = c.BuiltInParamBuilder(item)
        if err != nil {
            return fmt.Errorf("PostParamBuilder build Line[%s]getErr[%s]", item, err.Error()), ""
        }
        if param == nil { //skip
            return
        }
    } else {
        return errors.New("not find paramBuilder for GET req"), ""
    }

    var datajs []byte
    if reflect.TypeOf(param).Kind() == reflect.String {
        datajs = []byte(param.(string))
    } else {
        datajs, _ = sonic.Marshal(param)
    }

    if c.BuiltInParamAppender != nil {
        paramAppend, err := c.BuiltInParamAppender(item)
        if err != nil {
            return fmt.Errorf("BuiltInParamAppender Line[%s]getErr[%s]", item, err.Error()), ""
        }
        var dataMap map[string]interface{}
        json.Unmarshal(datajs, &dataMap)
        for s, i := range paramAppend {
            dataMap[s] = i
        }
        datajs, _ = json.Marshal(dataMap)
    }

    if c.DryRun {
        fmt.Println(item, "\n", string(datajs))
        return
    }
    payload := bytes.NewReader(datajs)
    req, err := http.NewRequest("POST", c.url, payload)
    if err != nil {
        return errors.New(fmt.Sprintf("http NewRequest err %s", err.Error())), ""
    }
    if c.Header != nil {
        for key, value := range c.Header {
            req.Header.Set(key, value)
        }
    }
    req.Header.Add("Content-Type", "application/json")
    if c.ReqHost != "" {
        req.Host = c.ReqHost
    }
    if c.cookieStr != "" {
        req.Header.Add("Cookie", c.cookieStr)
    }
    atomic.AddUint64(&c.reqTotal, 1)
    var res *http.Response
    for retry := c.Retry + 1; retry > 0; retry-- {
        res, err = DefaultHttpClient.Do(req)
        if err == nil {
            break
        }
        time.Sleep(1 * time.Second)
    }
    if err != nil {
        fmt.Println(string(datajs))
        atomic.AddUint64(&c.reqFail, 1)
        return errors.New(fmt.Sprintf("ClientDoReqErr%dtimes,err:%s", c.Retry+1, err.Error())), ""
    }
    defer res.Body.Close()
    if res.StatusCode != 200 {
        fmt.Println(item, "\n", string(datajs))
        atomic.AddUint64(&c.statusCodeErr, 1)
        return errors.New(fmt.Sprintf("StatusCode  %d", res.StatusCode)), ""
    }
    if c.DetailLog || !c.DiscardResBody {
        body, _ := ioutil.ReadAll(res.Body)
        bodys := string(body)
        if c.DetailLog {
            fmt.Println(item, "\n", string(datajs), "\n", bodys)
        }
        if c.ResErrNoName != "" {
            errInfo := gjson.Get(bodys, c.ResErrNoName).String()

            if errInfo != "0" {
                atomic.AddUint64(&c.parseBodyErr, 1)
                return errors.New(bodys), ""
            }
        }
        if c.SaveRes {
            if c.SaveResDataExtractor != "" {
                resData = gjson.Get(bodys, c.SaveResDataExtractor).String()
            } else {
                resData = bodys
            }
        }
    } else {
        io.Copy(ioutil.Discard, res.Body)
    }

    atomic.AddUint64(&c.succ, 1)
    return
}

func (c *apiPoster) itemGet(item string) (err error, resData string) {
    url := c.url
    if c.ParamDirect {
        url += item
    } else if c.GetParamTemplate != "" {
        if c.SrcFileColumNum == 1 {
            url = strings.Replace(url, "%s", item, -1)
        } else {
            arr := strings.Split(item, c.SrcFileColumSeparator)
            if len(arr) != c.SrcFileColumNum {
                err = errors.New("illegal srcFile line")
                return
            }
            for i := 0; i < c.SrcFileColumNum; i++ {
                url = strings.Replace(url, "%s", arr[i], 1)
            }

        }
    } else if c.GetParamTemplateV2 != "" {
        url, err = TemplateReplace(url, item, c.SrcFileColumSeparator)
        if err != nil {
            return err, ""
        }
    } else if c.BuiltInParamBuilder != nil {
        query, err2 := c.BuiltInParamBuilder(item)
        if err2 != nil {
            return fmt.Errorf("GetParamBuilder build Line[%s]getErr[%s]", item, err.Error()), ""
        }
        queryStr, ok := query.(string)
        if !ok {
            return errors.New("BuiltInParamBuilder illegal res type"), ""
        }
        url += queryStr
    } else {
        return errors.New("not find paramBuilder for GET req"), ""
    }

    if c.BuiltInParamAppender != nil {
        paramAppend, err := c.BuiltInParamAppender(item)
        if err != nil {
            return fmt.Errorf("BuiltInParamAppender Line[%s]getErr[%s]", item, err.Error()), ""
        }
        url, err = appendParamsToURL(paramAppend, url)
        if err != nil {
            return fmt.Errorf("BuiltInParamAppender err Line[%s]appendParamsToURL getErr[%s]", item, err.Error()), ""
        }
    }

    if c.DryRun {
        fmt.Println(item, "\n", url)
        return
    }
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return errors.New(fmt.Sprintf("http NewRequest err %s", err.Error())), ""
    }
    if c.cookieStr != "" {
        req.Header.Add("Cookie", c.cookieStr)
    }
    if c.Header != nil {
        for key, value := range c.Header {
            req.Header.Set(key, value)
        }
    }
    req.Host = c.Host
    atomic.AddUint64(&c.reqTotal, 1)
    t := time.Now().UnixNano()
    var res *http.Response
    for retry := c.Retry + 1; retry > 0; retry-- {
        res, err = DefaultHttpClient.Do(req)
        if err == nil {
            break
        }
        time.Sleep(1 * time.Second)
    }
    if err != nil {
        fmt.Println(url)
        atomic.AddUint64(&c.reqFail, 1)
        return errors.New(fmt.Sprintf("ClientDoReqErr%dtimes,err:%s", c.Retry+1, err.Error())), ""
    }
    atomic.AddUint64(&c.Time, uint64(time.Now().UnixNano()-t)/1e6)
    defer res.Body.Close()
    if res.StatusCode != 200 {
        atomic.AddUint64(&c.statusCodeErr, 1)
        return errors.New(fmt.Sprintf("StatusCode  %d", res.StatusCode)), ""
    }

    if c.DetailLog || !c.DiscardResBody {
        body, _ := ioutil.ReadAll(res.Body)
        bodys := string(body)
        if c.DetailLog {
            fmt.Println(item, "\n", url, "\n", bodys)
        }
        if c.ResErrNoName != "" {
            errInfo := gjson.Get(bodys, c.ResErrNoName).String()
            if errInfo != "0" {
                atomic.AddUint64(&c.parseBodyErr, 1)
                return errors.New(errInfo), ""
            }
        }
        if c.SaveRes {
            if c.SaveResDataExtractor != "" {
                resData = gjson.Get(bodys, c.SaveResDataExtractor).String()
            } else {
                resData = bodys
            }
        }
    } else {
        io.Copy(ioutil.Discard, res.Body)
    }

    atomic.AddUint64(&c.succ, 1)
    return
}

func (c *apiPoster) RunStatistic() {
    oldStatistic := statisticData{}
    template := "req:%d reqFail:%d statusCodeErr:%d contentErr:%d succ%d %d%% %dms"
    var oldFail, nowFail, nowRes, decRes, decFail, decSuccRate, decAvgTime uint64
    var totalInfo, decInfo string

    for c.reqTotal == 0 {
        time.Sleep(time.Duration(c.checkInterval))
    }
    for c.parseBodyErr+c.statusCodeErr+c.reqFail+c.succ == 0 {
        time.Sleep(time.Duration(c.checkInterval))
    }

    for !c.done {
        time.Sleep(time.Duration(c.checkInterval))
        now := c.statisticData
        oldFail = oldStatistic.parseBodyErr + oldStatistic.statusCodeErr + oldStatistic.reqFail
        nowFail = now.parseBodyErr + now.statusCodeErr + now.reqFail
        nowRes = nowFail + now.succ
        decRes = nowRes - (oldFail + oldStatistic.succ)
        decFail = nowFail - oldFail
        decSuccRate = 100 - decFail*100/(nowRes)
        if decRes != 0 {
            decAvgTime = (now.Time - oldStatistic.Time) / decRes
        }

        totalInfo = fmt.Sprintf(template, now.reqTotal, now.reqFail, now.statusCodeErr, now.parseBodyErr, now.succ, now.succ*100/nowRes, now.Time/nowRes)
        decInfo = fmt.Sprintf(template, now.reqTotal-oldStatistic.reqTotal, now.reqFail-oldStatistic.reqFail, now.statusCodeErr-oldStatistic.statusCodeErr, now.parseBodyErr-oldStatistic.parseBodyErr, now.succ-oldStatistic.succ, decSuccRate, decAvgTime)
        fmt.Printf("[STATISTIC] now_%d(%s) total(%s)\n", time.Now().Unix(), decInfo, totalInfo)
        oldStatistic = now
    }
}

func (c *apiPoster) Run() (err error) {

    if c.Statistic {
        go c.RunStatistic()
    }

    go func() {
        err = c.itemProducerRun()
        if err != nil {
            log.Println("itemProducerRun err: ", err)
        }
        close(c.ch)
    }()
    go func() {
        err = c.errItemSaverRun()
        if err != nil {
            log.Println("errItemSaverRun err: ", err)
        }
    }()
    if c.SaveRes {
        go func() {
            err = c.resItemSaverRun()
            if err != nil {
                log.Println("errResItemSaverRun err: ", err)
            }
        }()
    }
    c.wg.Add(c.WorkerCoroutineNum)
    var errMakeRes error
    var tmpRes string
    for i := 0; i < c.WorkerCoroutineNum; i++ {
        go func() {
            defer c.wg.Done()
            if c.isGetMethod {
                for item := range c.ch {
                    err, resData := c.itemGet(item)
                    if err != nil {
                        log.Printf("sendGetReqErr,line[%s]err[%s]", item, err.Error())
                        c.errItemCh <- item
                    } else {
                        if c.SaveRes {
                            if c.SaveResDataTemplate != "" {
                                resData = strings.Replace(c.SaveResDataTemplate, "$resDataExtract", resData, 1)
                                tmpRes, errMakeRes = TemplateReplace(resData, item, c.SrcFileColumSeparator)
                                if errMakeRes != nil {
                                    log.Println("ResTemplateMake Err,skip make Res,", errMakeRes)
                                } else {
                                    resData = tmpRes
                                }
                            }
                            c.resCh <- resData
                        }
                    }
                }
            } else {
                for item := range c.ch {
                    err, resData := c.itemPost(item)
                    if err != nil {
                        log.Printf("sendPostReqErr,line[%s]err[%s]", item, err.Error())
                        c.errItemCh <- item
                    } else {
                        if c.SaveRes {
                            if c.SaveResDataTemplate != "" {
                                resData = strings.Replace(c.SaveResDataTemplate, "$resDataExtract", resData, 1)
                                tmpRes, errMakeRes = TemplateReplace(resData, item, c.SrcFileColumSeparator)
                                if errMakeRes != nil {
                                    log.Println("ResTemplateMake Err,skip make Res,", errMakeRes)
                                } else {
                                    resData = tmpRes
                                }
                            }
                            c.resCh <- resData
                        }
                    }
                }
            }
        }()
    }
    c.wg.Wait()
    close(c.errItemCh)
    if c.SaveRes {
        close(c.resCh)
    }
    time.Sleep(time.Second * 1)
    return
}

func TemplateReplace(template, line, sep string) (string, error) {
    // 分割line字符串
    parts := strings.Split(line, sep)

    // 构造结果字符串
    var result strings.Builder

    // 遍历模板字符串
    for i := 0; i < len(template); i++ {
        if template[i] == '$' {
            jsonEncode := false

            if i+4 < len(template) && template[i+1:i+5] == "JSON" {
                jsonEncode = true
                i += 4
            }

            // 确保'$'后有字符
            if i+1 < len(template) && template[i+1] >= '0' && template[i+1] <= '9' {
                // 获取'$'后的数字
                num := 0
                j := i + 1
                for ; j < len(template) && template[j] >= '0' && template[j] <= '9'; j++ {
                    num = num*10 + int(template[j]-'0')
                }
                // 检查索引是否有效
                if num > len(parts) {
                    return "", fmt.Errorf("index %d out of range for line", num)
                }
                // 替换占位符
                var writeStr string
                if num == 0 {
                    writeStr = line
                } else {
                    writeStr = parts[num-1]
                }
                if jsonEncode {
                    writeStr, _ = sonic.MarshalString(writeStr)
                }
                result.WriteString(writeStr)
                // 更新i
                i = j - 1
            } else {
                // 如果'$'后没有字符，则原样输出'$'
                result.WriteByte(template[i])
            }
        } else {
            // 常规字符直接添加到结果中
            result.WriteByte(template[i])
        }
    }

    return result.String(), nil
}
