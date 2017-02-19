package main

import (
  //"bufio"
  "fmt"
  "net/http"
  //"log"
  "github.com/garyburd/redigo/redis"
  "encoding/json"
  "os/exec"
  "os"
  "strings"
  "io"
  "io/ioutil"
  "regexp"
  "strconv"
  "flag"
  "time"
  "encoding/binary"
  "crypto/rand"
)

var AESENC=""
var MODE=""
var AUTH=""
var DEBUG=""

func random() string {
    var n uint64
    binary.Read(rand.Reader, binary.LittleEndian, &n)
    return strconv.FormatUint(n, 36)
}

type Data struct {
        Title    string `json:"title"`
        Message  string `json:"message"`
        Status   string `json:"status"`
}

func validate(str string) string {
  str2 := str
  str2 = strings.Replace(str2,";","",-1)
  str2 = strings.Replace(str2,":","",-1)
  str2 = strings.Replace(str2,"&","",-1)
  str2 = strings.Replace(str2,"%","",-1)
  str2 = strings.Replace(str2,"\\","",-1)
  str2 = strings.Replace(str2,">","",-1)
  str2 = strings.Replace(str2,"<","",-1)
  str2 = strings.Replace(str2,"|","",-1)
  str2 = strings.Replace(str2,"!","",-1)
  str2 = strings.Replace(str2,"\"","",-1)
  str2 = strings.Replace(str2,"#","",-1)
  str2 = strings.Replace(str2,"$","",-1)
  str2 = strings.Replace(str2,"'","",-1)
  str2 = strings.Replace(str2,"(","",-1)
  str2 = strings.Replace(str2,")","",-1)
  //str2 = strings.Replace(str2,"*","",-1)
  str2 = strings.Replace(str2,"+","",-1)
  str2 = strings.Replace(str2,"=","",-1)
  str2 = strings.Replace(str2,"?","",-1)
  str2 = strings.Replace(str2,"@","",-1)
  str2 = strings.Replace(str2,"[","",-1)
  str2 = strings.Replace(str2,"]","",-1)
  str2 = strings.Replace(str2,"^","",-1)
  str2 = strings.Replace(str2,"`","",-1)
  str2 = strings.Replace(str2,"{","",-1)
  str2 = strings.Replace(str2,"}","",-1)
  return str2
}

func main() {
  _aesenc := flag.String("aes","","[-aes=(文字列)] 暗号化文字列(Default:password)")
  _mode := flag.String("mode","","[-mode=1,2,3] 1:Single(Default) 2:Remote(Require Auth) 3:Cluster(Require Auth)")
  _auth := flag.String("auth","","[-auth=(User_Key)...] 認証用のアカウント名と鍵ファイルを指定します")
  _port := flag.String("port","","[-port=(Port Number)] ポート番号(Default:1880)")
  _crt := flag.String("crt","","[-crt=(CRT File)] CRTファイルのパス(Default:./myself.crt)")
  _key := flag.String("key","","[-key=(KEY File)] KEYファイルのパス(Default:./myself.key)")
  dptr := flag.Bool("d", false, "[-d] debug mode")

  flag.Parse()

  AESENC = string(*_aesenc)
  if len(AESENC) < 4 || len(AESENC) > 20 {
    AESENC = "password"
  }
  PORT := ""
  portz,errz := strconv.Atoi(*_port)
  if portz < 0 || portz > 65535 || errz != nil {
    PORT = "1880"
  } else {
    PORT = string(*_port)
  }
  CRT := string(*_crt)
  if len(CRT) < 2 {
    CRT = "./myself.crt" 
  }
  KEY := string(*_key)
  if len(KEY) < 2 {
    KEY = "./myself.key"
  }

  if *dptr == true { 
    DEBUG = "1"
  } else {
    DEBUG = "0"
  }

  if DEBUG == "1" { fmt.Println("AES Encrypt: " + AESENC) }

  modea,_ := strconv.Atoi(string(*_mode))
  if modea < 1 || modea > 3 {
    MODE = "1"
  } else {
    MODE = validate(*_mode)
    AUTH = validate(*_auth)
    AUTH = strings.Replace(AUTH,"\n","",-1)
    if DEBUG == "1" { fmt.Println("MODE/AUTH: " + MODE + " " + AUTH) }
  }

  if MODE == "2" {
    rnd := random()
    dat1 := hostctl(2,AUTH,"ls && echo " + rnd)
    if DEBUG == "1" { fmt.Println("dat1: [" + dat1 + "]") }
    if strings.Index(dat1,rnd) == -1 {
      fmt.Println("Remote Mode Auth Fail: " + AUTH)
      os.Exit(1)
    }
  } else if MODE == "3" {
    dat1 := strings.Replace(AUTH,"~","\n",-1)
    if DEBUG == "1" { fmt.Println("dat1: [" + dat1 + "]") }
    for i, AUTH := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(dat1, -1) {
      if DEBUG == "1" { fmt.Println("Server: " +  strconv.Itoa(i+1) + " AUTH: [" + AUTH + "]") }
      rnd := random()
      dat2 := hostctl(2,AUTH,"ls && echo " + rnd)
      if DEBUG == "1" { fmt.Println("dat2: [" + dat2 + "]") }
        if strings.Index(dat2,rnd) == -1 {
        fmt.Println("Cluster Mode Auth Fail: " + AUTH)
        os.Exit(1)
      }
    }
    AUTH = dat1
  }

  outs := string(hostctl(1,AUTH,"ps -ef | grep \"ctl \" | grep -v grep | wc | tr -s \" \" | cut -d \" \" -f 2 | tr -d \"\\n\""))
  pss,_ := strconv.Atoi(outs)
  if pss > 1 {
    fmt.Println("Controller Duplicating: " + outs)
    os.Exit(1)
  }

  if DEBUG == "1" { fmt.Println("OPTIONS","password:" + AESENC,"mode:" + MODE,"AUTH:" + AUTH,"port: " + PORT,"crt: " + CRT,"key: " + KEY,"debug:" + DEBUG) }

  http.HandleFunc("/", handler)
  err := http.ListenAndServeTLS(":" + PORT, CRT, KEY, nil)
  if err != nil {
    fmt.Println("ListenAndServe: ", err)
    os.Exit(1)
  }
}

func getredis (auth string) string {
  world := ""

  c, err := redis.Dial("tcp", ":6379")
  if err != nil {
    return world
  }
  defer c.Close()

  world, err = redis.String(c.Do("GET", auth))
  if err != nil {
    return world
  }
  defer c.Close()

 return world
}

func chostctl(command string) string {
  result := ""
  dat1 := strings.Replace(AUTH,"~","\n",-1)
  if DEBUG == "1" { fmt.Println("Cluster Host Ctl: [" + dat1 + "]") }
  for _, AUTH := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(dat1, -1) {
    rnd := random()
    dat2 := hostctl(2,AUTH,command + " && echo " + rnd)
    if DEBUG == "1" { fmt.Println("Output: [" + dat2 + "]") }
      if strings.Index(dat2,rnd) == -1 {
      fmt.Println("Cluster Mode Command Fail: " + AUTH)
      result += "Cluster Mode Command Fail: " + AUTH
    }
  }
  return result
}

func ghostctl(mode int,auth,command string) string {
  rstring := ""

  c, _ := redis.Dial("tcp", ":6379")
  world, _ := redis.String(c.Do("GET", auth))
  c.Close()
  world = strings.Replace(world,",","\n",-1)
  if DEBUG == "1" { fmt.Println("group: " + world) }

  for _, gcnt := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(world, -1) {
    if (len(gcnt) > 1) {
      acnt := strings.Split(gcnt, "_")
      cmda := strings.Replace(command,"##GRUOP##",acnt[0],-1)

      if mode != 3 {
        rstring += " -- " + acnt[0] + " --\n" + hostctl(mode,AUTH,cmda)
      } else if mode == 4 {
        rstring += " -- " + acnt[0] + " --\n" + hostctl(4,AUTH,cmda)
      } else {
        rstring += " -- " + acnt[0] + " --\n" + hostctlc(1,cmda,acnt[0])
      }
    }
  }
  return rstring
}

func hostctl(mode int,auth,command string) string {
  // mode = 1 : local
  // mode = 2 : remote
  // mode = 3 : clustered
  out := []byte("")

  switch {
    case mode == 1:
      out, _ = exec.Command(os.Getenv("SHELL"), "-c", command + " 2>&1 | tee").Output()
      if DEBUG == "1" { fmt.Printf("local exec command :%s :%s\n", command,out) }
    case mode == 2:
      acnt := strings.Split(auth,",")
      cmds := "ssh -t -t -p " + acnt[3] + " -oStrictHostKeyChecking=no -i " + acnt[1] + " " + acnt[0] + "@" + acnt[2] + " '" + command +"'"
      out, _ = exec.Command(os.Getenv("SHELL"), "-c", cmds  + " 2>&1 | tee").Output()
      if DEBUG == "1" { fmt.Printf("remote exec command :%s :%s\n", cmds,out) }
    case mode == 3:
      bcnt := strings.Replace(AUTH,"~","\n",-1)
      outs := []byte("")
      for i, acct := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(bcnt, -1) {
        if DEBUG == "1" { fmt.Println("acct",i+1, ": ", acct) }
        acnt := strings.Split(acct,",")
        cmds := "ssh -t -t -p " + acnt[3] + " -oStrictHostKeyChecking=no -i " + acnt[1] + " " + acnt[0] + "@" + acnt[2] + " '" + command +"'"
        out, _ = exec.Command(os.Getenv("SHELL"), "-c", cmds  + " 2>&1 | tee").Output()
        if DEBUG == "1" { fmt.Printf("cluster exec command :%s :%s\n", cmds,out) }
        if i == 0 {
          outs = out
        } else {
          dat := strings.Replace(string(out),"\r\n","\t",-1)
          datt := strings.Split(dat,"\t")
          tt, _ := time.Parse("2006-01-02-15-04-05", datt[1])

          dat = strings.Replace(string(outs),"\r\n","\t",-1)
          datt = strings.Split(dat,"\t")
          ttt, _ := time.Parse("2006-01-02-15-04-05", datt[1])
          if tt.Unix() > ttt.Unix() {
            outs = out
            if DEBUG == "1" { fmt.Println("Replace: " + string(outs)) }
          }
        }
     }
     out = outs
    case mode == 4:
      bcnt := strings.Replace(AUTH,"~","\n",-1)
      rest := ""
      aacnt := ""
      for i, acct := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(bcnt, -1) {
        if DEBUG == "1" { fmt.Println("acct",i+1, ": ", acct) }
        acnt := strings.Split(acct,",")
        aacnt = acnt[2]
        cmds := "ssh -t -t -p " + acnt[3] + " -oStrictHostKeyChecking=no -i " + acnt[1] + " " + acnt[0] + "@" + acnt[2] + " '" + command +"'"
        out, _ = exec.Command(os.Getenv("SHELL"), "-c", cmds  + " 2>&1 | tee").Output()
        if DEBUG == "1" { fmt.Printf("cluster2 exec command :%s :%s\n", cmds,out) }
        rest += string(out)
      }
      rest = strings.Replace(rest,"\r","",-1)
      ioutil.WriteFile("/tmp/" + aacnt, []byte(rest), os.ModePerm)
      outs, _ := exec.Command(os.Getenv("SHELL"), "-c","sort /tmp/" + aacnt + " 2>&1 | uniq").Output()
      if DEBUG == "1" { fmt.Printf(string(outs)) }
      out = outs
      exec.Command(os.Getenv("SHELL"), "-c","rm -f /tmp/" + aacnt)
  }
  return strings.Replace(string(out),"\r","",-1)
}

func hostctlc(mode int,command,account string) string {
  // mode = 1 : local
  // mode = 2 : remote
  // mode = 3 : clustered
  out := []byte("")

  switch {
    case mode == 1:
      r := 0
      bcnt := strings.Replace(AUTH,"~","\n",-1)
      outs := []byte("")
      for i, acct := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(bcnt, -1) {
        if DEBUG == "1" { fmt.Println("Cluster Host:",i+1, " : ", acct) }
        acnt := strings.Split(acct,",")
        cmds := "ssh -t -t -p " + acnt[3] + " -oStrictHostKeyChecking=no -i " + acnt[1] + " " + acnt[0] + "@" + acnt[2] + " 'cat /home/" + account + "/status*'"
        out, _ = exec.Command(os.Getenv("SHELL"), "-c", cmds  + " 2>&1 | tee").Output()
        if DEBUG == "1" { fmt.Printf("cluster now check :%s :%s\n", cmds,out) }
        if i == 0 {
          outs = out
        } else {
          dat := strings.Replace(string(out),"\r\n","\t",-1)
          datt := strings.Split(dat,"\t")
          tt, _ := time.Parse("2006-01-02-15-04-05", datt[1])

          dat = strings.Replace(string(outs),"\r\n","\t",-1)
          datt = strings.Split(dat,"\t")
          ttt, _ := time.Parse("2006-01-02-15-04-05", datt[1])
          if tt.Unix() > ttt.Unix() {
            outs = out
            if DEBUG == "1" { fmt.Println("Replace: " + string(outs)) }
            r = i
          }
        }
     }
     if DEBUG == "1" { fmt.Println("Newer Host Number: " + strconv.Itoa(r+1)) }
     for i, acct := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(bcnt, -1) {
       acnt := strings.Split(acct,",")
       if i == r {
         cmds := "ssh -t -t -p " + acnt[3] + " -oStrictHostKeyChecking=no -i " + acnt[1] + " " + acnt[0] + "@" + acnt[2] + " '" + command +"'"
         out, _ = exec.Command(os.Getenv("SHELL"), "-c", cmds  + " 2>&1 | tee").Output()
         if DEBUG == "1" { fmt.Printf("cluster exec command :%s :%s\n", cmds,out) }
       }
     }
  }
  return strings.Replace(string(out),"\r","",-1)
}

func authac(mode int,account string) int {
  modea,_ := strconv.Atoi(MODE)

  if mode == 3 {
     account = strings.Replace(account,",","\n",-1)
     if DEBUG == "1" {  fmt.Println("group: " + account) }

     for _, gcnt := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(account,-1) {
       if (len(gcnt) > 1) {
         acnt := strings.Split(gcnt, "_")

         line := hostctl(modea,AUTH,"cat /home/" + acnt[0] + "/pswd")
         if len(line) < 1 { return 1 }
         lines := strings.Replace(string(line),"\n","",-1)
         acnt[1] = strings.Replace(acnt[1],"\n","",-1)
         if DEBUG == "1" { fmt.Println("lines:" + lines) }
         if DEBUG == "1" { fmt.Println("acnt:" + acnt[1]) }
         if acnt[1] != lines {
           return 1
         }
       }
     }
     return 0
  } else if mode == 2 {
     c, err := redis.Dial("tcp", ":6379")
     if err != nil {
       panic(err)
     }
     defer c.Close()

     world, _ := redis.String(c.Do("GET", account))
     if len(world)  < 1 { return 1 }
     world = strings.Replace(world,",","\n",-1)
     if DEBUG == "1" {  fmt.Println("group: " + world) }

     for _, gcnt := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(world,-1) {
       if (len(gcnt) > 1) {
         acnt := strings.Split(gcnt, "_")

         line := hostctl(modea,AUTH,"cat /home/" + acnt[0] + "/pswd")
         if len(line) < 1 { return 1 }
         lines := strings.Replace(string(line),"\n","",-1)
         acnt[1] = strings.Replace(acnt[1],"\n","",-1)
         if DEBUG == "1" { fmt.Println("lines:" + lines) }
         if DEBUG == "1" { fmt.Println("acnt:" + acnt[1]) }
         if acnt[1] != lines {
           return 1
         }
       }
     }
     return 0
  } else {
    acnt := strings.Split(account, "_")

    line := hostctl(modea,AUTH,"cat /home/" + acnt[0] + "/pswd")
    if len(line) < 1 { return 1 }
    lines := strings.Replace(string(line),"\n","",-1)
    aacnt := strings.Replace(acnt[1],"\n","",-1)
    if DEBUG == "1" { fmt.Println("lines: " + lines + " acnt: " + acnt[1]) } 
    if aacnt != lines {
      return 1
    }
    return 0
  }
}

func Exists(name string) bool {
    _, err := os.Stat(name)
    return !os.IsNotExist(err)
}

func handler(w http.ResponseWriter, r *http.Request) {
  var data = Data{}
  gmode := 0
  modea,_ := strconv.Atoi(MODE)

  if DEBUG == "1" { fmt.Println("method: ", r.Method) }
  if r.Method == "GET" {
    //Auth
    account := r.URL.Query()["account"]
    group := r.URL.Query()["group"]
    cmd := r.URL.Query()["cmd"]

    if cmd[0] != "init" {
      if len(group) > 0 {
        group[0] = validate(group[0])
        if authac(2,group[0]) != 0 {
          data.Title = "Group Auth"
          data.Message = group[0]
          data.Status = "fail"
          outputJson, _ := json.Marshal(&data)
          w.Header().Set("Content-Type", "application/json")
          fmt.Fprint(w, string(outputJson))
          if DEBUG == "1" { fmt.Println("Group Auth Wrong!: " + group[0]) }
          return
        }
        gmode = 1
      } else if len(account) > 0 {
        account[0] = validate(account[0])
        if authac(1,account[0]) != 0 {
          data.Title = "Account Auth"
          data.Message = account[0]
          data.Status = "fail"
          outputJson, _ := json.Marshal(&data)
          w.Header().Set("Content-Type", "application/json")
          fmt.Fprint(w, string(outputJson))
          if DEBUG == "1" { fmt.Println("Account Auth Wrong!: " + account[0]) }
          return
        } else {
          acnt := strings.Split(account[0], "_")
          account[0] = acnt[0]
        }
      } else {
        data.Title = "No Account"
        data.Message = ""
        data.Status = "fail"
        outputJson, _ := json.Marshal(&data)
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprint(w, string(outputJson))
        if DEBUG == "1" { fmt.Println("No Account!") }
        return
      }
    }

    if gmode == 1 { fmt.Println("Group Mode On") }

    urlstring := strings.Replace(r.URL.String(),"?","&",-1)
    if DEBUG == "1" { fmt.Println("url: " + urlstring) }

    if strings.Index(urlstring,"&cmd=") == -1 {
      data.Title = "No Command"
      data.Message = ""
      data.Status = "fail"
      outputJson, _ := json.Marshal(&data)
      w.Header().Set("Content-Type", "application/json")
      fmt.Fprint(w, string(outputJson))
      if DEBUG == "1" { fmt.Println("No Command!") }
      return
    }

    cmd[0] = validate(cmd[0])

    com := 0
    cmds := ""
    data.Title = cmd[0]
    if DEBUG == "1" { fmt.Println("Command: " + cmd[0]) }

    switch cmd[0] {
    case "status":
      com = 1
      data.Status = "ok"
      if gmode == 1 {
        cmds = "cat /home/##GRUOP##/status*"
      } else {
        cmds = "cat /home/" + account[0] + "/status*"
      }
    case "ruleshow":
      com = 2
      data.Status = "ok"
      if gmode == 1 {
        cmds = "cat /home/##GRUOP##/alert.conf"
      } else {
        cmds = "cat /home/" + account[0] + "/alert.conf"
      }
    case "groupget":
      com = 3
      if strings.Index(urlstring,"&params=") == -1 {
        data.Message = "No Params Set"
        data.Status = "fail"
      } else {
        params := r.URL.Query()["params"]
        params[0] = validate(params[0])
        result := getredis(params[0])
        if DEBUG == "1" { fmt.Println("Params: " + params[0] + " Result:" + result) }
        if len(result) > 1 {
          if gmode == 1 {
            data.Message = "Group Command Auth Not Allow Group"
            data.Status = "fail"
          } else {
            data.Message = params[0]
            data.Status = result
          }
        } else {
          data.Message = "Group Not Found"
          data.Status = "fail"
        }
      }
    case "groupset":
      com = 4
      if strings.Index(urlstring,"&params=") == -1 || strings.Index(urlstring,"&accounts=") == -1 {
        data.Message = "No Params or Accounts Set"
        data.Status = "fail"
      } else {
        params := r.URL.Query()["params"]
        accounts := r.URL.Query()["accounts"]
        params[0] = validate(params[0])
        accounts[0] = validate(accounts[0])
        if DEBUG == "1" { fmt.Println("Params: " + params[0]) }
        datt := strings.Split(params[0],"_")
        if len(datt[0]) < 4 || len(datt[1]) < 4 {
          data.Message = "Group Define Error: " + params[0] + ": 4 _ 4"
          data.Status = "fail"
        } else { 
          if gmode == 1 {
            data.Message = "Group Command Auth Not Allow Group"
            data.Status = "fail"
          } else {
            if authac(3,accounts[0]) != 0 {
              data.Message = "Auth Error: " + accounts[0]
              data.Status = "fail"
            } else {
              data.Message = accounts[0]
              data.Status = "ok"
              c, _ := redis.Dial("tcp", ":6379")
              c.Do("SET", params[0], accounts[0])
              c.Close()
              if DEBUG == "1" { fmt.Println("Grop Set",params[0], accounts[0]) }

              rest := getredis(account[0])
              if strings.Index(rest, datt[0] + ",") == -1 {
                if len(rest) > 1 {
                  rest += params[0] + ","
                } else {
                  rest = params[0] + ","
                }
                c, _ = redis.Dial("tcp", ":6379")
                c.Do("SET", account[0], rest)
                c.Close()
                if DEBUG == "1" { fmt.Println("Grop list",account[0],rest) }
              } else { 
                if DEBUG == "1" { fmt.Println("Grop list Exists",account[0],rest) }
              }
            }
          }
        }
      }
    case "grouplist":
      com = 5
      result := getredis(account[0])
      if DEBUG == "1" { fmt.Println("Account: " + account[0] + " Result:" + result) }
      if len(result) > 1 {
        if gmode == 1 {
          data.Message = "Group Command Auth Not Allow Group"
          data.Status = "fail"
        } else {
          data.Message = result
          data.Status = "ok"
        }
      } else {
        data.Message = "Group Not Found"
        data.Status = "fail"
      }
    case "groupdel":
      com = 6
      if strings.Index(urlstring,"&params=") == -1 {
        data.Message = "No Params Set"
        data.Status = "fail"
      } else {
        params := r.URL.Query()["params"]
        params[0] = validate(params[0])
        result := getredis(account[0])
        if DEBUG == "1" { fmt.Println("Params: " + params[0] + " Result:" + result) }
        if len(result) > 1 {
          if gmode == 1 {
            data.Message = "Group Command Auth Not Allow Group"
            data.Status = "fail"
          } else {
            if strings.Index(result,params[0] + ",") != -1 {
              result := strings.Replace(result,params[0] + ",","",-1)
              c, _ := redis.Dial("tcp", ":6379")
              c.Do("SET", account[0], result)
              c.Close()

              data.Message = "Group " + params[0] + " is deleted"
              data.Status = "ok"
            } else {
              data.Message = "Group " + params[0] +  " is not found"
              data.Status = "fail"
            }
          }
        } else {
          data.Message = "Account Group Not Found"
          data.Status = "fail"
        }
      }
    case "ruleset":
      com = 7
      if strings.Index(urlstring,"&params=") == -1 {
        data.Message = "No Params Set"
        data.Status = "fail"
      } else {
        params := r.URL.Query()["params"]
        params[0] = validate(params[0])
        datt := strings.Split(params[0], "_")
        rflag := 0
        if DEBUG == "1" { fmt.Println("Params: ",params[0],strconv.Itoa(len(datt))) }
        if len(datt) != 4 {
          rflag = 5
          data.Message = "Params Not Enough: " + strconv.Itoa(len(datt)) + " : 4"
          data.Status = "fail"
        } else {
          count,_ := strconv.Atoi(datt[3])
          if len(datt[0]) < 3 || len(datt[0]) > 20 {
            rflag = 1
            data.Message = "Rule Name Invalid: " + datt[0] + " : 3 _ 20"
            data.Status = "fail"
          } else if datt[1] != "d" && datt[1] != "D" && datt[1] != "s" && datt[1] != "S" {
            rflag = 2
            data.Message = "Check Type Invalid: " + datt[1] + " : d D s S"
            data.Status = "fail"
          } else if len(datt[2]) < 3 || len(datt[2]) > 20 {
            rflag = 3
            data.Message = "Check Parameter Invalid: " + datt[2] + " : 3 _ 20"
            data.Status = "fail"
          } else if count < 0 || count > 999 {
            rflag = 4
            data.Message = "Check Count Invalid: " + datt[3] + " : 0 _ 999"
            data.Status = "fail"
          }
        }
        wrule := ""
        rrflag := 0
        if rflag == 0 { 
          if gmode == 1 {
            result := getredis(group[0])
            if DEBUG == "1" { fmt.Println("Group: " + result) }
            dattn := strings.Replace(result,",","\n",-1)
            for _, AUTH := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(dattn, -1) {
              dap :=  strings.Split(AUTH,"_")
              rules := hostctl(modea,AUTH,"cat /home/" + dap[0] + "/alert.conf")
              if len(rules) < 6 {
                data.Message = "Rule Add: " + params[0]
                for _, rcnt := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(rules, -1) {
                  if (len(rcnt) > 1) {
                    erule := strings.Split(rcnt, "\t")
                    if erule[0] != datt[0] {
                      wrule += rcnt + "\n"
                      if DEBUG == "1" { fmt.Println("Not Exists Rule: ",rcnt) }
                    } else {
                      if DEBUG == "1" { fmt.Println("Replace Rule: ",rcnt) }
                      data.Message = "Rule Replace: " + params[0]
                      rrflag = 1
                    }
                  }
                }
                wrule += strings.Replace(params[0],"_","\t",-1)
                if DEBUG == "1" { fmt.Println("New Rules: ",wrule) }
              } else {
                wrule = strings.Replace(params[0],"_","\t",-1)
              }
              if modea == 3 {
                result := chostctl("echo -e \"" + wrule + "\" > /home/" + dap[0] + "/alert.conf")
                if rrflag == 0 {
                  result += chostctl("echo -e \"" + datt[0] + "\t" + datt[3] + "\" >> /home/" + dap[0] + "/alertcount")
                }
                if len(result) < 1 {
                  data.Status = "ok"
                } else {
                  data.Message = result
                  data.Status = "fail"
                }
              } else {
                hostctl(modea,AUTH,"echo -e \"" + wrule + "\" > /home/" + dap[0] + "/alert.conf")
                if rrflag == 0 {
                  hostctl(modea,AUTH,"echo -e \"" + datt[0] + "\t" + datt[3] + "\" >> /home/" + dap[0] + "/alertcount")
                }
                data.Status = "ok"
              }
            }
          } else {
            rules := hostctl(modea,AUTH,"cat /home/" + account[0] + "/alert.conf")
            wrule := ""
            rrflag := 0
            data.Message = "Rule Add: " + params[0]
            for _, rcnt := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(rules, -1) {
              if (len(rcnt) > 1) {
                erule := strings.Split(rcnt, "\t")
                if erule[0] != datt[0] {
                  wrule += rcnt + "\n"
                  if DEBUG == "1" { fmt.Println("Not Exists Rule: ",rcnt) }
                } else {
                  if DEBUG == "1" { fmt.Println("Replace Rule: ",rcnt) }
                  data.Message = "Rule Replace: " + params[0]
                  rrflag = 1
                }
              }
            }
            wrule += strings.Replace(params[0],"_","\t",-1)
            if DEBUG == "1" { fmt.Println("New Rules: ",wrule) }
            if modea == 3 {
              result := chostctl("echo -e \"" + wrule + "\" > /home/" + account[0] + "/alert.conf")
              if rrflag == 0 {
                result += chostctl("echo -e \"" + datt[0] + "\t" + datt[3] + "\" >> /home/" + account[0] + "/alertcount")
              }
              if len(result) < 1 {
                data.Status = "ok"
              } else {
                data.Message = result
                data.Status = "fail"
              }
            } else {
              hostctl(modea,AUTH,"echo -e \"" + wrule + "\" > /home/" + account[0] + "/alert.conf")
              if rrflag == 0 {
                hostctl(modea,AUTH,"echo -e \"" + datt[0] + "\t" + datt[3] + "\" >> /home/" + account[0] + "/alertcount")
              }
              data.Status = "ok"
            }
          }
        }
      }
    case "ruledel":
      com = 8
      if strings.Index(urlstring,"&params=") == -1 {
        data.Message = "No Params Set"
        data.Status = "fail"
      } else {
        params := r.URL.Query()["params"]
        params[0] = validate(params[0])
        if len(params[0]) < 3 || len(params[0]) > 20 {
          data.Message = "Rule Name Invalid: " + params[0] + " : 3 _ 20"
          data.Status = "fail"
        } else {
          if gmode == 1 {
            result := getredis(group[0])
            if DEBUG == "1" { fmt.Println("Group: " + result) }
            dattn := strings.Replace(result,",","\n",-1)
            for _, AUTH := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(dattn, -1) {
              dap :=  strings.Split(AUTH,"_")
              rules := hostctl(modea,AUTH,"cat /home/" + dap[0] + "/alert.conf")
              wrule := ""
              rrflag := 1
              data.Message = "Rule Delete: " + params[0]
              for _, rcnt := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(rules, -1) {
                if (len(rcnt) > 1) {
                  erule := strings.Split(rcnt, "\t")
                  if erule[0] != params[0] {
                    wrule += rcnt + "\n"
                  } else {
                    rrflag = 0
                  }
                }
              }
              if DEBUG == "1" { fmt.Println("New Rules: ",wrule) }
              if modea == 3 {
                result := ""
                if rrflag == 0 {
                  result = chostctl("echo -e \"" + wrule + "\" > /home/" + dap[0] + "/alert.conf")
                  result += chostctl("sed -i '/" + params[0] + "\t[0-999]/d' /home/" + dap[0] + "/alertcount")
                  data.Status = "ok"
                } else {
                  data.Message = "Rule Not Found: " + params[0]
                  data.Status = "fail"
                }
                if len(result) > 0 {
                  data.Message = result
                  data.Status = "fail"
                }
              } else {
                if rrflag == 0 {
                  hostctl(modea,AUTH,"echo -e \"" + wrule + "\" > /home/" + dap[0] + "/alert.conf")
                  hostctl(modea,AUTH,"sed -i '/" + params[0] + "\t[0-999]/d' /home/" + dap[0] + "/alertcount")
                  data.Status = "ok"
                } else {
                  data.Message = "Rule Not Found: " + params[0]
                  data.Status = "fail"
                }
              }
            }
          } else {
            rules := hostctl(modea,AUTH,"cat /home/" + account[0] + "/alert.conf")
            wrule := ""
            rflag := 1
            data.Message = "Rule Delete: " + params[0]
            for _, rcnt := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(rules, -1) {
              if (len(rcnt) > 1) {
                erule := strings.Split(rcnt, "\t")
                if erule[0] != params[0] {
                  wrule += rcnt + "\n" 
                } else {
                  rflag = 0
                }
              }
            }
            if DEBUG == "1" { fmt.Println("New Rules: ",wrule) }
            if modea == 3 {
              result := ""
              if rflag == 0 {
                result = chostctl("echo -e \"" + wrule + "\" > /home/" + account[0] + "/alert.conf")
                result += chostctl("sed -i '/" + params[0] + "\t[0-999]/d' /home/" + account[0] + "/alertcount")
                data.Status = "ok"
              } else {
                data.Message = "Rule Not Found: " + params[0]
                data.Status = "fail"
              }
              if len(result) > 0 {
                data.Message = result
                data.Status = "fail"
              }
            } else {
              if rflag == 0 {
                hostctl(modea,AUTH,"echo -e \"" + wrule + "\" > /home/" + account[0] + "/alert.conf")
                hostctl(modea,AUTH,"sed -i '/" + params[0] + "\t[0-999]/d' /home/" + account[0] + "/alertcount")
                data.Status = "ok"
              } else {
                data.Message = "Rule Not Found: " + params[0]
                data.Status = "fail"
              }
            }
          }
        }
      }
    case "agentget":
      com = 9
    case "commit":
      com = 10
      if strings.Index(urlstring,"&params=") == -1 || strings.Index(urlstring,"&to=") == -1 {
        data.Message = "No Params or To Set"
        data.Status = "fail"
      } else {
        params := r.URL.Query()["params"]
        to := r.URL.Query()["to"]
        params[0] = validate(params[0])
        to[0] = validate(to[0])
        rflag := 0
        if len(params[0]) < 3 || len(params[0]) > 41 {
          data.Message = "Rule Name Invalid: " + params[0] + " : 3 _ 20"
          data.Status = "fail"
          rflag = 1
        }
        if DEBUG == "1" { fmt.Println("to: " + to[0]) }
        if to[0] != "alert" && to[0] != "action" && to[0] != "result" {
          data.Message = "to is Invalid: " + to[0] + " : alert or action or result"
          data.Status = "fail"
          rflag = 2
        }
        commitf := strings.Split(params[0], "_")
        commr := commitf[0]
        commitz := ""
        if to[0] == "action" || to[0] == "result" {
          commitz = "0_" + params[0]
          commitf[0] = "0_" + commitf[0]
          commr = "0_" + commr
        } else {
          commitz = params[0]
        }
        if rflag == 0 {
          if gmode == 1 {
            result := getredis(group[0])
            if DEBUG == "1" { fmt.Println("Group: " + result) }
            dattn := strings.Replace(result,",","\n",-1)
            for _, AUTH := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(dattn, -1) {
              dap :=  strings.Split(AUTH,"_")
              rnd := random()
              dat2 := hostctl(modea,AUTH,"ls /home/" + dap[0] + "/" + to[0] + "/repo_" + commitz +" && echo " + rnd)
              if DEBUG == "1" { fmt.Println("Output: [" + dat2 + "]") }
              if strings.Index(dat2,rnd) != -1 {
                if modea == 3 {
                  chostctl("rm -f /home/" + dap[0] + "/" + to[0] + "/" + commr + "_*")
                  result := chostctl("TMPCKSM=`cksum /home/" + dap[0] + "/" + to[0] + "/repo_" + commitz + " | tr -s \" \" | cut -d \" \" -f 1 | tr -d \"\\n\"` ; cd /home/" + dap[0] + "/" + to[0] + "; ln -s repo_" + commitz + " " + commitf[0] + "_${TMPCKSM}")
                  if len(result) < 1 {
                    data.Message = "Commit Sccess: " + params[0]
                    data.Status = "ok"
                  } else {
                    data.Message = "Commit not found: " + result
                    data.Status = "fail"
                  }
                } else {
                  hostctl(modea,AUTH,"rm -f /home/" + dap[0] + "/" + to[0] + "/" + commr + "_*")
                  hostctl(modea,AUTH,"TMPCKSM=`cksum /home/" + dap[0] + "/" + to[0] + "/repo_" + commitz + " | tr -s \" \" | cut -d \" \" -f 1 | tr -d \"\\n\"` ; cd /home/" + dap[0] + "/" + to[0] + "; ln -s repo_" + commitz + " " + commitf[0] + "_${TMPCKSM}")
                  data.Message = "Commit Sccess: " + params[0]
                  data.Status = "ok"
                }
              } else {
                data.Message = "Commit not found: " + params[0]
                data.Status = "fail"
              }
            }
          } else {
            rnd := random()
            dat2 := hostctl(modea,AUTH,"ls /home/" + account[0] + "/" + to[0] + "/repo_" + commitz +" && echo " + rnd)
            if DEBUG == "1" { fmt.Println("Output: [" + dat2 + "]") }
            if strings.Index(dat2,rnd) != -1 {
              if modea == 3 {
                chostctl("rm -f /home/" + account[0] + "/" + to[0] + "/" + commr + "_*")
                result := chostctl("TMPCKSM=`cksum /home/" + account[0] + "/" + to[0] + "/repo_" + commitz + " | tr -s \" \" | cut -d \" \" -f 1 | tr -d \"\\n\"` ; cd /home/" + account[0] + "/" + to[0] + "; ln -s repo_" + commitz + " " + commitf[0] + "_${TMPCKSM}")
                if len(result) < 1 {
                  data.Message = "Commit Sccess: " + params[0]
                  data.Status = "ok"
                } else {
                  data.Message = "Commit not found: " + result
                  data.Status = "fail"
                }
              } else {
                if DEBUG == "1" { fmt.Println(commr) }
                hostctl(modea,AUTH,"rm -f /home/" + account[0] + "/" + to[0] + "/" + commr + "_*")
                hostctl(modea,AUTH,"TMPCKSM=`cksum /home/" + account[0] + "/" + to[0] + "/repo_" + commitz + " | tr -s \" \" | cut -d \" \" -f 1 | tr -d \"\\n\"` ; cd /home/" + account[0] + "/" + to[0] + "; ln -s repo_" + commitz + " " + commitf[0] + "_${TMPCKSM}")
                data.Message = "Commit Sccess: " + params[0]
                data.Status = "ok"
              }
            } else {
              data.Message = "Commit not found: " + params[0]
              data.Status = "fail"
            }
          }
        }
      }
    case "uncommit":
      com = 11
      if strings.Index(urlstring,"&params=") == -1 || strings.Index(urlstring,"&to=") == -1 {
        data.Message = "No Params or To Set"
        data.Status = "fail"
      } else {
        params := r.URL.Query()["params"]
        to := r.URL.Query()["to"]
        params[0] = validate(params[0])
        to[0] = validate(to[0])
        rflag := 0
        if len(params[0]) < 3 || len(params[0]) > 20 {
          data.Message = "Rule Name Invalid: " + params[0] + " : 3 _ 20"
          data.Status = "fail"
          rflag = 1
        }
        if DEBUG == "1" { fmt.Println("to: " + to[0]) }
        if to[0] != "alert" && to[0] != "action" && to[0] != "result" {
          data.Message = "to is Invalid: " + to[0] + " : alert or action or result"
          data.Status = "fail"
          rflag = 2
        }
        commitf := strings.Split(params[0], "_")
        if to[0] == "action" || to[0] == "result" {
          commitf[0] = "0_" + commitf[0]
        } else {
        }
        if rflag == 0 {
          if gmode == 1 {
            result := getredis(group[0])
            if DEBUG == "1" { fmt.Println("Group: " + result) }
            dattn := strings.Replace(result,",","\n",-1)
            for _, AUTH := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(dattn, -1) {
              dap :=  strings.Split(AUTH,"_")
              rnd := random()
              dat2 := hostctl(modea,AUTH,"ls /home/" + dap[0] + "/" + to[0] + "/" + params[0] +"_* && echo " + rnd)
              if DEBUG == "1" { fmt.Println("Output: [" + dat2 + "]") }
              if strings.Index(dat2,rnd) != -1 {
                if modea == 3 {
                  result := chostctl("TMPRULE=`ls /home/" + dap[0] + "/" + to[0] + "/" + params[0] + "_* | tr -d \"\\n\"` ; unlink ${TMPRULE}")
                  if len(result) < 1 {
                    data.Message = "UnCommit Sccess: " + params[0]
                    data.Status = "ok"
                  } else {
                    data.Message = "UnCommit not found: " + result
                    data.Status = "fail"
                  }
                } else {
                  hostctl(modea,AUTH,"TMPRULE=`ls /home/" + dap[0] + "/" + to[0] + "/" + params[0] + "_* | tr -d \"\\n\"` ; unlink ${TMPRULE}")
                  data.Message = "UnCommit Sccess: " + params[0]
                  data.Status = "ok"
                }
              } else {
                data.Message = "UnCommit not found: " + params[0]
                data.Status = "fail"
              }
            }
          } else {
            rnd := random()
            dat2 := hostctl(modea,AUTH,"ls /home/" + account[0] + "/" + to[0] + "/" + params[0] +"_* && echo " + rnd)
            if DEBUG == "1" { fmt.Println("Output: [" + dat2 + "]") }
            if strings.Index(dat2,rnd) != -1 {
              if modea == 3 {
                result := chostctl("TMPRULE=`ls /home/" + account[0] + "/" + to[0] + "/" + params[0] + "_* | tr -d \"\\n\"` ; unlink ${TMPRULE}")
                if len(result) < 1 {
                  data.Message = "UnCommit Sccess: " + params[0]
                  data.Status = "ok"
                } else {
                  data.Message = "UnCommit not found: " + result
                  data.Status = "fail"
                }
              } else {
                hostctl(modea,AUTH,"TMPRULE=`ls /home/" + account[0] + "/" + to[0] + "/" + params[0] + "_* | tr -d \"\\n\"` ; unlink ${TMPRULE}")
                data.Message = "UnCommit Sccess: " + params[0]
                data.Status = "ok"
              }
            } else {
              data.Message = "UnCommit not found: " + params[0]
              data.Status = "fail"
            }
          }
        }
      }
    case "repolist":
      com = 12
      if strings.Index(urlstring,"&to=") == -1 {
        data.Message = "No To Set"
        data.Status = "fail"
      } else {
        to := r.URL.Query()["to"]
        to[0] = validate(to[0])
        rflag := 0
        if DEBUG == "1" { fmt.Println("to: " + to[0]) }
        if to[0] != "alert" && to[0] != "action" && to[0] != "result" {
          data.Message = "to is Invalid: " + to[0] + " : alert or action or result"
          data.Status = "fail"
          rflag = 2
        }
        if rflag == 0 {
          data.Status = "ok"
          if gmode == 1 {
            if to[0] == "action" || to[0] == "result" {
              cmds = "cd /home/##GRUOP##/" + to[0] + " ; ls repo_*_* | sed -e \"s/repo_0_//g\""
            } else {
              cmds = "cd /home/##GRUOP##/" + to[0] + " ; ls repo_*_* | sed -e \"s/repo_//g\""
            }
          } else {
            if to[0] == "action" || to[0] == "result" {
              cmds = "cd /home/" + account[0] + "/" + to[0] + " ; ls repo_*_* | sed -e \"s/repo_0_//g\""
            } else {
              cmds = "cd /home/" + account[0] + "/" + to[0] + " ; ls repo_*_* | sed -e \"s/repo_//g\""
            }
          }
        }
      }
    case "commitcat":
      com = 13
      if strings.Index(urlstring,"&params=") == -1 || strings.Index(urlstring,"&to=") == -1 {
        data.Message = "No Params or To Set"
        data.Status = "fail"
      } else {
        params := r.URL.Query()["params"]
        to := r.URL.Query()["to"]
        params[0] = validate(params[0])
        to[0] = validate(to[0])
        rflag := 0
        if len(params[0]) < 3 || len(params[0]) > 20 {
          data.Message = "Rule Name Invalid: " + params[0] + " : 3 _ 20"
          data.Status = "fail"
          rflag = 1
        }
        if DEBUG == "1" { fmt.Println("to: " + to[0]) }
        if to[0] != "alert" && to[0] != "action" && to[0] != "result" {
          data.Message = "to is Invalid: " + to[0] + " : alert or action or result"
          data.Status = "fail"
          rflag = 2
        }
        commitf := strings.Split(params[0], "_")
        if to[0] == "action" || to[0] == "result" {
          commitf[0] = "0_" + commitf[0]
        } 
        if rflag == 0 {
          data.Status = "ok"
          if gmode == 1 {
            cmds = "cd /home/##GRUOP##/" + to[0] + " ; ls -l " + commitf[0] + "_* | sed \"s/.*-> repo_//g\" | tr -d \"\n\" ; printf \": \" ; cat " + commitf[0] + "_*"
          } else {
            cmds = "cd /home/" + account[0] + "/" + to[0] + " ; ls -l " + commitf[0] + "_* | sed \"s/.*-> repo_//g\" | tr -d \"\n\" ; printf \": \" ; cat " + commitf[0] + "_*"
          }
        }
      }
    case "repocat":
      com = 14
      if strings.Index(urlstring,"&params=") == -1 || strings.Index(urlstring,"&to=") == -1 {
        data.Message = "No Params or To Set"
        data.Status = "fail"
      } else {
        params := r.URL.Query()["params"]
        to := r.URL.Query()["to"]
        params[0] = validate(params[0])
        to[0] = validate(to[0])
        rflag := 0
        if len(params[0]) < 3 || len(params[0]) > 41 {
          data.Message = "Rule Name Invalid: " + params[0] + " : 3 _ 20"
          data.Status = "fail"
          rflag = 1
        }
        if DEBUG == "1" { fmt.Println("to: " + to[0]) }
        if to[0] != "alert" && to[0] != "action" && to[0] != "result" {
          data.Message = "to is Invalid: " + to[0] + " : alert or action or result"
          data.Status = "fail"
          rflag = 2
        }
        commitf := "repo_"
        if to[0] == "action" || to[0] == "result" {
          commitf += "0_"
        }
        commitf += params[0] 
        if rflag == 0 {
          data.Status = "ok"
          if gmode == 1 {
            cmds = "cat /home/##GRUOP##/" + to[0] + "/" + commitf
          } else {
            cmds = "cat /home/" + account[0] + "/" + to[0] + "/" + commitf
          }
        }
      }
    case "repodel":
      com = 15
      if strings.Index(urlstring,"&params=") == -1 || strings.Index(urlstring,"&to=") == -1 {
        data.Message = "No Params or To Set"
        data.Status = "fail"
      } else {
        params := r.URL.Query()["params"]
        to := r.URL.Query()["to"]
        params[0] = validate(params[0])
        to[0] = validate(to[0])
        rflag := 0
        if len(params[0]) < 3 || len(params[0]) > 41 {
          data.Message = "Rule Name Invalid: " + params[0] + " : 3 _ 20"
          data.Status = "fail"
          rflag = 1
        }
        if DEBUG == "1" { fmt.Println("to: " + to[0]) }
        if to[0] != "alert" && to[0] != "action" && to[0] != "result" {
          data.Message = "to is Invalid: " + to[0] + " : alert or action or result"
          data.Status = "fail"
          rflag = 2
        }
        commitf := "repo_"
        if to[0] == "action" || to[0] == "result" {
          commitf += "0_"
        }
        commitf += params[0]
        result := ""
        if gmode == 1 {
          cmds = "cat /home/##GRUOP##/" + to[0] + "/" + commitf
          result = ghostctl(modea,group[0],cmds)
          re := regexp.MustCompile(" -- .* --\n")
          result = re.ReplaceAllString(result,"")
        } else {
          cmds = "cat /home/" + account[0] + "/" + to[0] + "/" + commitf
          if modea == 3 {
            result = hostctlc(1,cmds,account[0])
          } else {
            result = hostctl(modea,AUTH,cmds)
          }
        }
        if len(result) < 1 || strings.Index(result,"No such file") != -1 {
          data.Message = "Rule Not Found: " + params[0]
          data.Status = "fail"
          rflag = 3
        }
        if rflag == 0 {
          data.Message = params[0]
          data.Status = "ok"
          if gmode == 1 {
            cmds = "rm /home/##GRUOP##/" + to[0] + "/" + commitf
          } else {
            cmds = "rm /home/" + account[0] + "/" + to[0] + "/" + commitf
          }
        }
      }
    case "init":
      com = 16
      rflag := 0
      if strings.Index(urlstring,"&password=") == -1 || strings.Index(urlstring,"&params=") == -1 {
        data.Message = "No Server Password or Params Set"
        data.Status = "fail"
        rflag = 1
      } else {
        params := r.URL.Query()["params"]
        params[0] = validate(params[0])
        datt := strings.Split(params[0], "_")
        password := r.URL.Query()["password"]
        password[0] = validate(password[0])
        if password[0] != AESENC {
          data.Message = "Server Password Wrong"
          data.Status = "fail"
          rflag = 2
        }
        if len(datt) != 2 {
          data.Message = "Params Not Enough: " + strconv.Itoa(len(datt)) + " : 2"
          data.Status = "fail"
         rflag = 3
        } else {
          if len(datt[0]) < 3 || len(datt[0]) > 20 {
            data.Message = "Account Name Invalid: " + datt[0] + " : 3 _ 20"
            data.Status = "fail"
            rflag = 4
          }
          if len(datt[1]) < 3 || len(datt[1]) > 20 {
            data.Message = "Account Password Invalid: " + datt[1] + " : 3 _ 20"
            data.Status = "fail"
            rflag = 5
          }
        }
        if gmode == 1 {
          data.Message = "Group Command Auth Not Allow Group"
          data.Status = "fail"
          rflag = 6
        } 
        if rflag == 0 {
          if modea == 3 {
            result := ""
            sshkey := ""
            sshpub := ""
            AUTHBK := AUTH
            dat1 := strings.Replace(AUTH,"~","\n",-1)
            for i, AUTH := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(dat1, -1) {
              if i == 0 {
                result = hostctl(2,AUTH,"grep " + datt[0] + ": /etc/passwd")
                if len(result) < 1 {
                  data.Message += "Host: " + AUTH + " Account: " + datt[0] + " Password: " + datt[1] + " Created"
                  hostctl(2,AUTH,"useradd " + datt[0] + " -d /home/" + datt[0] + " ; mkdir -p /home/" + datt[0] + " ; chown " + datt[0] + ":" + datt[0] + " /home/" + datt[0])
                  hostctl(2,AUTH,"su - " + datt[0] + " -c \"ssh-keygen -f ~/.ssh/id_rsa -t rsa -N \\\"\\\"\"")
                  hostctl(2,AUTH,"su - " + datt[0] + " -c \"cat ~/.ssh/id_rsa.pub >> ~/.ssh/authorized_keys ; chmod 0600 ~/.ssh/authorized_keys ; mkdir ~/action ~/alert ~/in ~/out ~/metric ~/result ; touch ~/alert.conf ; touch ~/alertcount\"")
                  hostctl(2,AUTH,"echo \"" + datt[1] + "\" >> /home/" + datt[0] + "/pswd")
                  sshkey = hostctl(2,AUTH,"cat /home/" + datt[0] + "/.ssh/id_rsa")
                  sshpub = hostctl(2,AUTH,"su - " + datt[0] + " -c \"cat ~/.ssh/id_rsa.pub\"")
                  data.Status = sshkey
                } else {
                  data.Message += "Host: " + AUTH + "Account: " + datt[0] + " Exits. not create"
                  data.Status = "fail"
                }
              } else {
                result := hostctl(2,AUTH,"grep " + datt[0] + ": /etc/passwd")
                if len(result) < 1 {
                  data.Message += "Host: " + AUTH + " Account: " + datt[0] + " Password: " + datt[1] + " Created"
                  hostctl(2,AUTH,"useradd " + datt[0] + " -d /home/" + datt[0] + " ; mkdir -p /home/" + datt[0] + " ; chown " + datt[0] + ":" + datt[0] + " /home/" + datt[0])
                  hostctl(2,AUTH,"su - " + datt[0] + " -c \"mkdir ~/.ssh ~/action ~/alert ~/in ~/out ~/metric ~/result ; touch ~/alert.conf ; touch ~/alertcount\"")
                  hostctl(2,AUTH,"echo \"" + datt[1] + "\" >> /home/" + datt[0] + "/pswd")
                  hostctl(2,AUTH,"su - " + datt[0] + " -c \"echo -e \\\"" + sshpub + "\\\" >> ~/.ssh/authorized_keys ; chmod 600 ~/.ssh/authorized_keys ; chmod 700 ~/.ssh\"")
                } else {
                  data.Message += "Host: " + AUTH + " Account: " + datt[0] + " Exits. not create"
                  data.Status = "fail"
                }
              }
            }
            AUTH = AUTHBK
          } else if modea == 2 {
            result := hostctl(modea,AUTH,"grep " + datt[0] + ": /etc/passwd")
            if len(result) < 1 {
              data.Message = "Account: " + datt[0] + " Password: " + datt[1] + " Created"
              hostctl(modea,AUTH,"useradd " + datt[0] + " -d /home/" + datt[0] + " ; mkdir -p /home/" + datt[0] + " ; chown " + datt[0] + ":" + datt[0] + " /home/" + datt[0])
              hostctl(modea,AUTH,"su - " + datt[0] + " -c \"ssh-keygen -f ~/.ssh/id_rsa -t rsa -N \\\"\\\"\"")
              hostctl(modea,AUTH,"su - " + datt[0] + " -c \"cat ~/.ssh/id_rsa.pub >> ~/.ssh/authorized_keys ; chmod 600 ~/.ssh/authorized_keys ; mkdir ~/action ~/alert ~/in ~/out ~/metric ~/result ; touch ~/alert.conf ; touch ~/alertcount\"")
              hostctl(modea,AUTH,"echo \"" + datt[1] + "\" >> /home/" + datt[0] + "/pswd")
              result := hostctl(modea,AUTH,"cat /home/" + datt[0] + "/.ssh/id_rsa")
              data.Status = result
            } else {
              data.Message = "Account: " + datt[0] + " Exits. not create"
              data.Status = "fail"
            }
          } else if modea == 1 {
            result := hostctl(modea,AUTH,"grep " + datt[0] + ": /etc/passwd")
            if len(result) < 1 {
              data.Message = "Account: " + datt[0] + " Password: " + datt[1] + " Created"
              hostctl(modea,AUTH,"useradd " + datt[0] + " -d /home/" + datt[0] + " ; mkdir -p /home/" + datt[0] + " ; chown " + datt[0] + ":" + datt[0] + " /home/" + datt[0])
              hostctl(modea,AUTH,"su -c 'ssh-keygen -f ~/.ssh/id_rsa -t rsa -N \"\"' " + datt[0] + " ; su -c 'cat ~/.ssh/id_rsa.pub >> ~/.ssh/authorized_keys' " + datt[0] + " ; su -c 'chmod 600 ~/.ssh/authorized_keys ; mkdir ~/action ~/alert ~/in ~/out ~/metric ~/result' " + datt[0] + " ; su -c 'touch ~/alert.conf' " + datt[0] + " ; su -c 'touch ~/alertcount' " + datt[0] + " ; echo \"" + datt[1] + "\" >> /home/" + datt[0] + "/pswd")
              result := hostctl(modea,AUTH,"cat /home/" + datt[0] + "/.ssh/id_rsa")
              data.Status = result
            } else {
              data.Message = "Account: " + datt[0] + " Exits. not create"
              data.Status = "fail"
            }
          }
        }
      }
    case "repocopy":
      com = 17
      rflag := 0
      if strings.Index(urlstring,"&to=") == -1 {
        data.Message = "To Not Set"
        data.Status = "fail"
        rflag = 1
      } else {
        to := r.URL.Query()["to"]
        to[0] = validate(to[0])
        datt := strings.Split(to[0], "_")
        if len(datt) != 2 {
          data.Message = "To Not Enough: " + strconv.Itoa(len(datt)) + " : 2"
          data.Status = "fail"
          rflag = 2
        } else {
          if len(datt[0]) < 3 || len(datt[0]) > 20 {
            data.Message = "Account Name Invalid: " + datt[0] + " : 3 _ 20"
            data.Status = "fail"
            rflag = 4
          }
          if len(datt[1]) < 3 || len(datt[1]) > 20 {
            data.Message = "Account Password Invalid: " + datt[1] + " : 3 _ 20"
            data.Status = "fail"
            rflag = 5
          }
        }
        if gmode == 1 {
          data.Message = "Group Command Auth Not Allow Group"
          data.Status = "fail"
          rflag = 6
        }
        if rflag == 0 {
          if modea == 3 {
            result := ""
            AUTHBK := AUTH
            dat1 := strings.Replace(AUTH,"~","\n",-1)
            for _, AUTH := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(dat1, -1) {
              result = hostctl(2,AUTH,"cat /home/" + datt[0] + "/pswd | tr -d \"\\n\"")
              if DEBUG == "1" { fmt.Println("Host: " + AUTH + " " + datt[0] + "/" + datt[1] + " : " + result) }
              if datt[1] == result {
                data.Message += "Host: " + AUTH + " Copy Account: " + account[0] + " to " + datt[0] + " Success"
                data.Status = "ok"
                hostctl(2,AUTH,"rm -rf /home/" + datt[0] + "/action")
                hostctl(2,AUTH,"mkdir /home/" + datt[0] + "/action")
                hostctl(2,AUTH,"cp -ad /home/" + account[0] + "/action/* /home/" + datt[0] + "/action")

                hostctl(2,AUTH,"rm -rf /home/" + datt[0] + "/alert")
                hostctl(2,AUTH,"mkdir /home/" + datt[0] + "/alert")
                hostctl(2,AUTH,"cp -ad /home/" + account[0] + "/alert/* /home/" + datt[0] + "/alert")

                hostctl(2,AUTH,"rm -rf /home/" + datt[0] + "/result")
                hostctl(2,AUTH,"mkdir /home/" + datt[0] + "/result")
                hostctl(2,AUTH,"cp -ad /home/" + account[0] + "/result/* /home/" + datt[0] + "/result")

                hostctl(2,AUTH,"cp -f /home/" + account[0] + "/alert.conf /home/" + datt[0] + "/alert.conf")
                hostctl(2,AUTH,"cat /home/" + account[0] + "/alert.conf | cut -f 1,4 > /home/" + datt[0] + "/alertcount")
                hostctl(2,AUTH,"chown -R " + datt[0] + ":" + datt[0] + " /home/" + datt[0])
                //hostctl(2,AUTH,"su - " + datt[0] + "; su -c 'rm -rf ./action ./alert ./result'; su -c 'mkdir ./action ./alert ./result' ; su -c 'cp -ad /home/" + account[0] + "/alert/* /home/" + datt[0] + "/alert' ; su -c 'cp -f /home/" + account[0] + "/alert.conf .' ; su -c 'chown -R " + datt[0] + ":" + datt[0] + " /home/" + datt[0] + "'")
              } else {
                data.Message += "Host: " + AUTH + " Account: " + datt[0] + " Auth fail"
                data.Status = "fail"
              }
            }
            AUTH = AUTHBK
          } else {
            data.Message = "Copy Account: " + account[0] + " to " + datt[0] + " Success"
            data.Status = "ok"

            result := hostctl(modea,AUTH,"cat /home/" + datt[0] + "/pswd | tr -d \"\\n\"")
            if DEBUG == "1" { fmt.Println(datt[0] + "/" + datt[1] + " : " + result) }
            if datt[1] == result {
              hostctl(modea,AUTH,"rm -rf /home/" + datt[0] + "/action")
              hostctl(modea,AUTH,"mkdir /home/" + datt[0] + "/action")
              hostctl(modea,AUTH,"cp -ad /home/" + account[0] + "/action/* /home/" + datt[0] + "/action")

              hostctl(modea,AUTH,"rm -rf /home/" + datt[0] + "/alert")
              hostctl(modea,AUTH,"mkdir /home/" + datt[0] + "/alert")
              hostctl(modea,AUTH,"cp -ad /home/" + account[0] + "/alert/* /home/" + datt[0] + "/alert")

              hostctl(modea,AUTH,"rm -rf /home/" + datt[0] + "/result")
              hostctl(modea,AUTH,"mkdir /home/" + datt[0] + "/result")
              hostctl(modea,AUTH,"cp -ad /home/" + account[0] + "/result/* /home/" + datt[0] + "/result")

              hostctl(modea,AUTH,"cp -f /home/" + account[0] + "/alert.conf /home/" + datt[0] + "/alert.conf")
              hostctl(modea,AUTH,"cat /home/" + account[0] + "/alert.conf | cut -f 1,4 > /home/" + datt[0] + "/alertcount")
              hostctl(modea,AUTH,"chown -R " + datt[0] + ":" + datt[0] + " /home/" + datt[0])
              //hostctl(modea,AUTH,"su - " + datt[0] + "; su -c 'rm -rf ./action ./alert ./result'; su -c 'mkdir ./action ./alert ./result' ; su -c 'cp -ad /home/" + account[0] + "/alert/* /home/" + datt[0] + "/alert' ; su -c 'cp -f /home/" + account[0] + "/alert.conf .' ; su -c 'chown -R " + datt[0] + ":" + datt[0] + " /home/" + datt[0] + "'")
            } else {
              data.Message = "Account: " + datt[0] + " Auth fail"
              data.Status = "fail"
            }
          }
        }
      }
    case "metric":
      com = 18
      if strings.Index(urlstring,"&params=") == -1 || strings.Index(urlstring,"&to=") == -1 {
        data.Message = "No Params or To Set"
        data.Status = "fail"
      } else {
        params := r.URL.Query()["params"]
        to := r.URL.Query()["to"]
        params[0] = validate(params[0])
        to[0] = validate(to[0])
        rflag := 0
        if len(to[0]) < 3 || len(to[0]) > 20 {
          data.Message = "Rule Name Invalid: " + to[0] + " : 3 _ 20"
          data.Status = "fail"
          rflag = 1
        }
        datt := strings.Split(params[0], "_")
        if datt[0] == "C" || datt[0] == "T" {
          if datt[0] == "T" && len(datt) != 3 {
            data.Message = "Params Not Enough or (T)ime Set Error: " + strconv.Itoa(len(datt)) + " : 3 " + params[0]
            data.Status = "fail"
            rflag = 2
          } else if datt[0] == "C" && len(datt) != 2 {
            data.Message = "Params Not Enough or (C)ount Set Error: " + strconv.Itoa(len(datt)) + " : 2 " + params[0]
            data.Status = "fail"
            rflag = 2
          } else {
            if rflag == 0 {
              data.Status = "ok"
              if gmode == 1 {
                bkmodea := modea
                if modea == 3 { modea = 4 }
                if datt[0] == "C" {
                  if DEBUG == "1" { fmt.Println(datt[0] + "/" + datt[1]) }
                  data.Status = "ok"
                  result := ghostctl(modea,group[0],"find /home/##GRUOP##/metric/" + to[0] + "/* -mtime " + datt[1] + " -type f | while read -r file; do TMP=`date -r ${file} && printf \"_\" && cat ${file} | tr -d \"\\n\"` && echo $TMP; done")
                  re := regexp.MustCompile(" -- .* --\n")
                  resultz := re.ReplaceAllString(result,"")
                  if len(resultz) < 1 {
                    data.Message = "Search Error: " + to[0] + " / " + params[0]
                    data.Status = "fail"
                  } else {
                    data.Message = result
                  }
                } else  {
                  if DEBUG == "1" { fmt.Println(datt[0] + "/" + datt[1] + ":" + datt[2]) }
                  data.Status = "ok"
                  result := ghostctl(modea,group[0],"find /home/##GRUOP##/metric/" + to[0] + "/* -newermt '" + datt[1] + "' -and ! -newermt '" + datt[2] +  "' -type f | while read -r file; do TMP=`date -r ${file} && printf \"_\" && cat ${file} | tr -d \"\\n\"` && echo $TMP; done")
                  re := regexp.MustCompile(" -- .* --\n")
                  resultz := re.ReplaceAllString(result,"")
                  if len(resultz) < 1 {
                    data.Message = "Search Error: " + to[0] + " / " + params[0]
                    data.Status = "fail"
                  } else {
                    data.Message = result
                  }
                }
                modea = bkmodea
              } else {
                bkmodea := modea
                if modea == 3 { modea = 4 }
                if datt[0] == "C" {
                  data.Status = "ok"
                  data.Message = hostctl(modea,AUTH,"find /home/" + account[0] + "/metric/" + to[0] + "/* -mtime " + datt[1] + " -type f | while read -r file; do TMP=`date -r ${file} && printf \"_\" && cat ${file} | tr -d \"\\n\"` && echo $TMP; done")
                  if len(data.Message) < 1 {
                    data.Message = "Search Error: " + to[0] + " / " + params[0]
                    data.Status = "fail"
                  }
                } else  {
                  if DEBUG == "1" { fmt.Println(datt[0] + "/" + datt[1] + ":" + datt[2]) }
                  data.Status = "ok"
                  data.Message = hostctl(modea,AUTH,"find /home/" + account[0] + "/metric/" + to[0] + "/* -newermt '" + datt[1] + "' -and ! -newermt '" + datt[2] +  "' -type f | while read -r file; do TMP=`date -r ${file} && printf \"_\" && cat ${file} | tr -d \"\\n\"` && echo $TMP; done")
                  if len(data.Message) < 1 {
                    data.Message = "Search Error: " + to[0] + " / " + params[0]
                    data.Status = "fail"
                  }
                }
                modea = bkmodea
              }
            }
          }
        } else {
          data.Message = "Params Not (C)ount or (T)imeSet Error: " + params[0]
          data.Status = "fail"
          rflag = 2
        }
      }
    default: 
      data.Status = "command not found"
      if DEBUG == "1" { fmt.Println("Command not found!") }
    }

    //if DEBUG == "1" { fmt.Println(modea,AUTH,cmds) }
    if com > 0 {
      switch {
      case com == 1:
        if gmode == 1 {
          data.Message = ghostctl(modea,group[0],cmds)
        } else {
          data.Message = hostctl(modea,AUTH,cmds)
        }
        if len(data.Message) < 1 || strings.Index(data.Message,"No such file") != -1 {
          data.Message = "Status Empty"
          data.Status = "fail"
        }
      case com == 2:
        if gmode == 1 {
          data.Message = ghostctl(modea,group[0],cmds)
        } else {
          if modea == 3 { 
            data.Message = hostctlc(1,cmds,account[0]) 
          } else { 
            data.Message = hostctl(modea,AUTH,cmds)
          }
        }
        if len(data.Message) < 1 || strings.Index(data.Message,"No such file") != -1 {
          data.Message = "Rule Empty"
          data.Status = "fail"
        }
      case com == 3:
      case com == 4:
      case com == 5:
      case com == 6:
      case com == 7:
      case com == 8:
      case com == 9:
        if DEBUG == "1" { fmt.Println("Agent Download") }
        ff, _ := os.Open("./agent")
        raw, _ := ioutil.ReadAll(ff)
        w.WriteHeader(200)
        w.Header().Set("Content-Type", "application/octet-stream")
        w.Write(raw)
        return
      case com == 10:
      case com == 11:
      case com == 12:
        if gmode == 1 {
          data.Message = ghostctl(modea,group[0],cmds)
        } else {
          if modea == 3 {
            data.Message = hostctlc(1,cmds,account[0])
          } else {
            data.Message = hostctl(modea,AUTH,cmds)
          }
        }
      case com == 13:
        if gmode == 1 {
          data.Message = ghostctl(modea,group[0],cmds)
        } else {
          if modea == 3 {
            data.Message = hostctlc(1,cmds,account[0])
          } else {
            data.Message = hostctl(modea,AUTH,cmds)
          }
        }
        if len(data.Message) < 1 || strings.Index(data.Message,"No such file") != -1 {
          data.Message = "Not Commit"
          data.Status = "fail"
        }
      case com == 14:
        if gmode == 1 {
          data.Message = ghostctl(modea,group[0],cmds)
        } else {
          if modea == 3 {
            data.Message = hostctlc(1,cmds,account[0])
          } else {
            data.Message = hostctl(modea,AUTH,cmds)
          }
        }
        if len(data.Message) < 1 || strings.Index(data.Message,"No such file") != -1 {
          data.Message = "Not Found or Empty"
          data.Status = "fail"
        }
      case com == 15:
        if gmode == 1 {
          result := getredis(group[0])
          if DEBUG == "1" { fmt.Println("Group: " + result) }
          dattn := strings.Replace(result,",","\n",-1)
          for _, AUTH := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(dattn, -1) {
            dap :=  strings.Split(AUTH,"_")
            cmda := strings.Replace(cmds,"##GRUOP##",dap[0],-1)
            if modea == 3 {
              chostctl(cmda)
            } else {
              hostctl(modea,AUTH,cmda)
            }  
          }
        } else {
          if modea == 3 {
            chostctl(cmds)
          } else {
            hostctl(modea,AUTH,cmds)
          }
        }
      case com == 16:
      case com == 17:
      case com == 18:
      default:
        //if gmode == 1 {
        //  data.Message = ghostctl(modea,group[0],cmds)
        //} else {
        //  data.Message = hostctl(modea,AUTH,cmds)
        //}
      }
    }

    outputJson, _ := json.Marshal(&data)
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprint(w, string(outputJson))
    return
  } else {
    r.ParseMultipartForm(32 << 20)
    data.Title = "Upload"
 
    //Auth
    account := r.FormValue("account")
    group := r.FormValue("group")

    if len(group) > 0 {
      group = validate(group)
      if authac(2,group) != 0 {
        data.Title = "Group Auth"
        data.Message = group
        data.Status = "fail"
        outputJson, _ := json.Marshal(&data)
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprint(w, string(outputJson))
        if DEBUG == "1" { fmt.Println("Group Auth Wrong!: " + group) }
        return
      }
      gmode = 1
    } else if len(account) > 0 {
      account = validate(account)
      if authac(1,account) != 0 {
        data.Title = "Account Auth"
        data.Message = account
        data.Status = "fail"
        outputJson, _ := json.Marshal(&data)
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprint(w, string(outputJson))
        if DEBUG == "1" { fmt.Println("Account Auth Wrong!: " + account) }
        return
      } else {
        acnt := strings.Split(account, "_")
        account = acnt[0]
      }
    } else {
      data.Title = "No Account"
      data.Message = ""
      data.Status = "fail"
      outputJson, _ := json.Marshal(&data)
      w.Header().Set("Content-Type", "application/json")
      fmt.Fprint(w, string(outputJson))
      if DEBUG == "1" { fmt.Println("No Account!") }
      return
    }

    rflag := 0
    to := r.FormValue("to")
    rulename := r.FormValue("rulename")
    if len(rulename) < 3 || len(rulename) > 20 {
      data.Message = "Rule Name Invalid: " + rulename + " : 3 _ 20"
      data.Status = "fail"
      rflag = 1
    }
    if to != "action" && to != "alert" && to != "result" {
      data.Message = "To is Invalid: " + to + " : action or alert or result"
      data.Status = "fail"
      rflag = 2
    }

    rulenamez := ""
    if to == "action" || to == "result" {
      rulenamez = "0_" + rulename
    } else {
      rulenamez = rulename 
    }

    if gmode == 1 { fmt.Println("Group Mode On") }    

    if rflag == 0 {
      if gmode == 1 {
        file, _, err := r.FormFile("file")
        if err != nil {
          data.Message = "Upload File Error"
          data.Status = "fail"
        } else {
          defer file.Close()
          //fmt.Fprintf(w, "%v", handler.Header)
          rnd := random()
          f, err := os.OpenFile("/tmp/" + rnd, os.O_WRONLY|os.O_CREATE, 0600)
          if err != nil {
            data.Message = "Upload Not Create"
            data.Status = "fail"
          } else {
            defer f.Close()
            io.Copy(f, file)
            fdat := hostctl(1,AUTH,"cat /tmp/" + rnd)
            if DEBUG == "1" { fmt.Println("File: " + fdat) }
            result := getredis(group)
            if DEBUG == "1" { fmt.Println("Group: " + result) }
            dattn := strings.Replace(result,",","\n",-1)
            cksm := hostctl(1,AUTH,"date +\"%Y-%m-%d-%I-%M-%S\" -r /tmp/" + rnd + " | tr -d \"\\n\"")
            for _, AUTH := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(dattn, -1) {
              dap :=  strings.Split(AUTH,"_")
              if modea == 3 {
                chostctl("echo -e \"" + fdat + "\" > /home/" + dap[0] + "/" + to + "/TMP_" + rulenamez)
                if DEBUG == "1" { fmt.Println("File: " + fdat) }
                chostctl("mv /home/" + dap[0] + "/" + to + "/TMP_" + rulenamez + " /home/" + dap[0] + "/" + to + "/repo_" + rulenamez + "_" + cksm)
              } else {
                hostctl(modea,AUTH,"echo -e \"" + fdat + "\" > /home/" + dap[0] + "/" + to + "/TMP_" + rulenamez)
                if DEBUG == "1" { fmt.Println("File: " + fdat) }
                hostctl(modea,AUTH,"mv /home/" + dap[0] + "/" + to + "/TMP_" + rulenamez + " /home/" + dap[0] + "/" + to + "/repo_" + rulenamez + "_" + cksm)
              }
            }
            data.Message = "To: " + to + " Rulename: " + rulename + "_" + cksm
            data.Status = "ok"
            hostctl(1,AUTH,"rm -f /tmp/" + rnd)
          }
        }
      } else {
        file, _, err := r.FormFile("file")
        if err != nil {
          data.Message = "Upload File Error"
          data.Status = "fail"
        } else {
          defer file.Close()
          //fmt.Fprintf(w, "%v", handler.Header)
          rnd := random()
          f, err := os.OpenFile("/tmp/" + rnd, os.O_WRONLY|os.O_CREATE, 0600)
          if err != nil {
            data.Message = "Upload Not Create"
            data.Status = "fail"
          } else {
            if modea == 3 {
              defer f.Close()
              io.Copy(f, file)
              fdat := hostctl(1,AUTH,"cat /tmp/" + rnd)
              cksm := hostctl(1,AUTH,"date +\"%Y-%m-%d-%I-%M-%S\" -r /tmp/" + rnd + " | tr -d \"\\n\"")
              chostctl("echo -e \"" + fdat + "\" > /home/" + account + "/" + to + "/TMP_" + rulenamez)
              if DEBUG == "1" { fmt.Println("File: " + fdat) }
              chostctl("mv /home/" + account + "/" + to + "/TMP_" + rulenamez + " /home/" + account + "/" + to + "/repo_" + rulenamez + "_" + cksm)
              data.Message = "To: " + to + " Rulename: " + rulename + "_" + cksm
              data.Status = "ok"
              hostctl(1,AUTH,"rm -f /tmp/" + rnd)
            } else {
              defer f.Close()
              io.Copy(f, file)
              fdat := hostctl(1,AUTH,"cat /tmp/" + rnd)
              cksm := hostctl(1,AUTH,"date +\"%Y-%m-%d-%I-%M-%S\" -r /tmp/" + rnd + " | tr -d \"\\n\"")
              hostctl(modea,AUTH,"echo -e \"" + fdat + "\" > /home/" + account + "/" + to + "/TMP_" + rulenamez)
              if DEBUG == "1" { fmt.Println("File: " + fdat) }
              hostctl(modea,AUTH,"mv /home/" + account + "/" + to + "/TMP_" + rulenamez + " /home/" + account + "/" + to + "/repo_" + rulenamez + "_" + cksm)
              data.Message = "To: " + to + " Rulename: " + rulename + "_" + cksm
              data.Status = "ok"
              hostctl(1,AUTH,"rm -f /tmp/" + rnd)
            }
          }
        }
      }
    }
    //data.Message = group
    //data.Status = "fail"
    outputJson, _ := json.Marshal(&data)
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprint(w, string(outputJson))
  }
}
