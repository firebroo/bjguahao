package main

import (
    "fmt"
    "net/http"
    "io/ioutil"
    "net/http/cookiejar"
    "net/url"
    "strings"
    "encoding/json"
    "strconv"
    "time"
    "os"
)

var (
    hospitalId string = "142"          /*医院id*/
    departmentId string = "200039602"  /*科室id*/
    dutyDate string = "2017-03-22"     /*挂号时间*/
    patientId string = "228398308"     /*病患id*/

    mobileNo string = "xxx"    /*手机号码*/
    password string = "xxx"      /*密码*/
)

type Doctor struct {
    DutySourceId int
    Portrait string
    DoctorName string
    DoctorTitleName string
    Skill string
    TotalFee float64
    RemainAvailableNumber int
    DutySourceStatus int
    HospitalId int
    DepartmentId string
    DoctorId string
    DrCode string
    PlanCode string
    DutyDate string
    DutyCode string
    DepartmentName string
}

type DoctorList struct {
    Data []Doctor
    HasError bool
    Code  int
    Msg string
}

type MsgRet struct {
    Code int
    Msg string
}

type RegisterStatus int

/*
@Luck 有号
@Nothing 没有号了
@Maybe 号还没有放出
*/
const (
    Luck = 0
    Maybe = 1
    Nothing = 2
)


func InitCookieClient() *http.Client {
    cookieJar, _ := cookiejar.New(nil)
    client := &http.Client{Jar: cookieJar, Timeout: 1 * time.Second}
    return client
}

func HttpDo(method string, url string, client *http.Client, data url.Values) string {
    req, _ := http.NewRequest(method, url, ioutil.NopCloser(strings.NewReader(data.Encode())))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    LABEL:
        resp, err := client.Do(req)
        if err != nil || resp.StatusCode == 403 {
            goto LABEL
        }

    defer resp.Body.Close()
                  
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {}
 
    return fmt.Sprintln(string(body))
}

func AuthLogin(client *http.Client) interface{} {
    ret := HttpDo("POST", "http://www.bjguahao.gov.cn/quicklogin.htm", client, 
            url.Values{
                "mobileNo": {mobileNo}, 
                "password": {password}, 
                "yzm": {""}, 
                "isAjax": {"true"},
            })

    var dat map[string]interface{}
    if err := json.Unmarshal([]byte(ret), &dat); err != nil {
        panic(err)
    }
    if dat["msg"] == "OK" { return nil } else { return dat["msg"] }
}

func (doctor Doctor)String() string {
    return fmt.Sprintf("医生: %s\t擅长: %s\t号余量: %d", doctor.DoctorName, doctor.Skill, doctor.RemainAvailableNumber)
}

func GreatPrintDoctorInfo(doctors []Doctor) {
    for _, doctor := range (doctors) {
        fmt.Println(doctor)
    }
}

func GetIds(client *http.Client) (map[string]string, RegisterStatus) {
    ret := HttpDo("POST", "http://www.bjguahao.gov.cn/dpt/partduty.htm", client, 
            url.Values{
                "hospitalId": {hospitalId},
                "departmentId": {departmentId},
                "dutyCode": {"1"},
                "dutyDate": {dutyDate},
                "isAjax": {"true"},
            })
    var dat DoctorList
    if err := json.Unmarshal([]byte(ret), &dat); err != nil {
        panic(err)
    }
    if data := dat.Data; len(data) == 0 {
        return map[string]string{}, Maybe
    } else {
        /* 尝试最后一个号，专家号 */
        lastDoctor := data[len(data)-1]
        if lastDoctor.RemainAvailableNumber != 0 {
            fmt.Println(lastDoctor)
            return map[string]string{"dutySourceId": strconv.Itoa(lastDoctor.DutySourceId), "doctorId": lastDoctor.DoctorId}, Luck
        }

        /* 尝试遍历挂号 */
        for _,doctor := range data {
            if doctor.RemainAvailableNumber != 0 {
                fmt.Println(doctor)
                return map[string]string{"dutySourceId": strconv.Itoa(doctor.DutySourceId), "doctorId": doctor.DoctorId}, Luck
            }
        }

        /* 没有号了，打印一下今天的号 */
        GreatPrintDoctorInfo(data)
    }

    return map[string]string{}, Nothing
}

func Register(client *http.Client, ids map[string]string, patientId string, msgCode string) {
    ret := HttpDo("POST", "http://www.bjguahao.gov.cn/order/confirm.htm", client,
            url.Values{
                "dutySourceId": {ids["dutySourceId"]},
                "hospitalId": {hospitalId},
                "departmentId": {departmentId},
                "doctorId": {ids["doctorId"]},
                "patientId": {patientId},
                "hospitalCardId": {""},
                "medicareCardId": {""},
                "reimbursementType": {"1"},
                "smsVerifyCode": {msgCode},
                "childrenBirthda": {""},
                "isAjax": {"true"},
            })
    var dat map[string]interface{}
    if err := json.Unmarshal([]byte(ret), &dat); err != nil {
        panic(err)
    }
    if dat["msg"] == "OK" {
        log("恭喜，挂号成功:)")
    } else {
        fmt.Println(dat["msg"])
        os.Exit(-1)
    }

}

func PeekDepartmentPage(client *http.Client) {
    url := fmt.Sprintf("http://www.bjguahao.gov.cn/dpt/appoint/%s-%s.htm", hospitalId, departmentId) 
    HttpDo("GET", url, client,  nil)
}

func PeekDetailsPage(client *http.Client, ids map[string]string) {
    url := fmt.Sprintf("http://www.bjguahao.gov.cn/order/confirm/%s-%s-%s-%s.htm", 
                      hospitalId, departmentId, ids["doctorId"], ids["dutySourceId"]) 
    HttpDo("GET", url, client,  nil)
}

func SendMsgCode(client *http.Client) interface{} {
    ret := HttpDo("POST", "http://www.bjguahao.gov.cn/v/sendorder.htm", client, url.Values{})
    var dat MsgRet
    if err := json.Unmarshal([]byte(ret), &dat); err != nil {
        panic(err)
    }
    if dat.Code == 200 && dat.Msg ==  "OK."  {
        return nil
    } else {
        return dat.Msg
    }
}

func SendMsgCodeAndGetMsgCode(client *http.Client) string {
    if err := SendMsgCode(client); err == nil {
        log("短信验证码发送成功...")
    } else {
        fmt.Println("error: ", err)
        os.Exit(-1)
    }
    var msgCode string
    fmt.Printf("%s: %s", time.Now().Format("2006-01-02 15:04:05"), "请输入验证码: ")
    fmt.Scanf("%s", &msgCode)
    return msgCode
}

func log(content string) {
    fmt.Printf("%s: %s\n", time.Now().Format("2006-01-02 15:04:05"), content)
}

func main() {
    client := InitCookieClient()
    if err := AuthLogin(client); err == nil {
        log("登陆成功...")
    } else {
        fmt.Println("error: ", err)
        os.Exit(-1)
    }
    go PeekDepartmentPage(client)

    var ids map[string]string
    var status RegisterStatus
    for {
        ids, status = GetIds(client)
        if status == Nothing {
            log("今天没有号了")
            os.Exit(-1)
        } else if status == Maybe {
            log("号还没有放出，等待...")
            time.Sleep(1e+09)
        } else {
            break;
        }
    }
    PeekDetailsPage(client, ids)
    msgCode := SendMsgCodeAndGetMsgCode(client)
    Register(client, ids, patientId, msgCode)
}
