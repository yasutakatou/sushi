package main

import (
    //"io"
    "os/exec"
    "os"
    "log"
    "fmt"
    "io/ioutil"
    "regexp"
    "strings"
    "strconv"
    "time"
    "flag"
    "bufio"
)

var LOCK=""

func main() {
  _timeout := flag.String("timeout","","[-timeout=(タイムアウト秒)] Agentからの通信途絶時間")
  lptr := flag.Bool("l", false, "[-l] no lock mode **WARNING!**")

  flag.Parse()

  timeout,_ := strconv.Atoi(string(*_timeout))
  if timeout < 10 {
    timeout = 60
  }
  fmt.Println("timeout: " + strconv.Itoa(timeout))

  if *lptr == true {
    LOCK = "1"
    fmt.Println("No Lock Mode! **WARNING**")
  } else {
    LOCK = "0"
  }


  status := ""

  fmt.Println(string(execmd("ps -ef | grep \"sv \"")))
  outs := string(execmd("ps -ef | grep \"sv \" | grep \"\\-timeout\" | grep -v grep | grep -v \"timeout \" | wc | tr -s \" \" | cut -d \" \" -f 2 | tr -d \"\\n\""))
  pss,_ := strconv.Atoi(outs)
  if pss > 1 {
    fmt.Println("Server Duplicating: " + outs)
    os.Exit(1)
  }

  statusout := []byte("")
  acclist := string(execmd("ls -l /home | grep -v \"total \" | tr -s \" \" | cut -d \" \" -f 9"))
  fmt.Println("Accounts: " + acclist)

  for i, acct := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(acclist, -1) {
    status = ""
    fmt.Println("acct",i+1, ": ", acct)

    if Exists("/home/" + acct + "/alert.conf") == true {
      if LOCK  == "1" { execmd("touch /home/" + acct + "/out/_lock_") }
      t := time.Now()
      const layout = "2006-01-02-15-04-05"
      fmt.Println(t.Format(layout))

      statusout = execmd("a=`cat /home/" + acct + "/alert.conf` ; b=`ls -l /home/" + acct + "/action` ; c=`ls -l /home/" + acct + "/alert` ; d=`ls -l /home/" + acct + "/result` ; echo $a$b$c$d | cksum | cut -d \" \" -f 1 | tr -d \"\\n\"")
      fmt.Println(string(statusout))
      if Exists("/home/" + acct + "/status_" + string(statusout)) == true {
        file, _ := os.OpenFile("/home/" + acct + "/out/_stay_",os.O_WRONLY|os.O_CREATE, 0755)
        file.Close()
        inlist := string(execmd("ls /home/" + acct + "/in/*"))
        fmt.Println("--")
        fmt.Println(string(inlist))
        fmt.Println("--")
        if len(inlist) > 10 && strings.Index(inlist,"No such file") == -1 {
          status += "Alive\t" + t.Format(layout) + "\n"
        } else {
          f,err := os.Open("/home/" + acct + "/status_" + string(statusout))
          fmt.Println("/home/" + acct + "/status_" + string(statusout))
          if err != nil {
            log.Fatal(err)
            return
          }
          r := bufio.NewReader(f)
          line,_ := r.ReadString('\n')
          f.Close()
          if len(line) > 5 { 
            datt := strings.Split(line,"\t")
            dattt := strings.Replace(datt[1],"\n","",-1)
            fmt.Println(dattt + " " + string(timeout))
            tt, _ := time.Parse("2006-01-02-15-04-05", dattt)
            fmt.Println(tt)
            fmt.Println(tt.Unix())
            fmt.Println(tt.Unix() + int64(timeout))
            fmt.Println(time.Now())
            fmt.Println(time.Now().Unix()  + 32400)
            if time.Now().Unix() + 32400 > (tt.Unix() + int64(timeout)) {
              status += "Disconnect\t" + datt[1] + "\n"
            } else {
              status += "Alive\t" + dattt + "\n"
            }
          } else {
            status += "Disconnect\t" + t.Format(layout) + "\n"
          }
        }
        fmt.Println("Rule Not Change: " + string(statusout))
      } else {
        execmd("rm -f /home/" + acct + "/status_*")
        file, _ := os.OpenFile("/home/" + acct + "/status_" + string(statusout), os.O_WRONLY|os.O_CREATE, 0755)
        file.Close()
        status += "Alive\t" + t.Format(layout) + "\n"
        fmt.Println("Rule Change: " + string(statusout))
        execmd("rm -f /home/" + acct + "/out/* ; ")
        execmd("rsync --copy-links -a /home/" + acct + "/alert/* /home/" + acct + "/out --exclude \"repo_*\"")
        execmd("rsync --copy-links -a /home/" + acct + "/action/* /home/" + acct + "/out --exclude \"repo_*\"")
        execmd("rsync --copy-links -a /home/" + acct + "/resultt/* /home/" + acct + "/out --exclude \"repo_*\"")
      }
      fmt.Println("status: " + status)

      buff, err := ioutil.ReadFile("/home/" + acct + "/alert.conf")
      if err != nil {
          fmt.Fprintln(os.Stderr, err)
          os.Exit(1)
      }
      fmt.Println(" -- acct rule -- ")
      os.Stdout.Write(buff)
      fmt.Println(" ---- ")

      cuff, err := ioutil.ReadFile("/home/" + acct + "/alertcount")
      if err != nil {
          fmt.Fprintln(os.Stderr, err)
          os.Exit(1)
      }
      fmt.Println(" -- acct count -- ")
      os.Stdout.Write(cuff)
      fmt.Println(" ---- ")
      aacnt := ""

      for i, rule := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(string(buff), -1) {
        if (len(rule) > 5) {
          fmt.Println("rule",i+1, ": ", rule)
          rules := strings.Split(rule, "\t")
          fmt.Println(rules[0] + ":" + rules[1] + ":" + rules[2])
          acctal := string(execmd("ls /home/" + acct + "/in/" + rules[0] + "* | head -1 | tr -d \"\\n\""))

          if Exists(acctal) == true {
            fmt.Println("exits file: " + acctal)

            flag, err := ioutil.ReadFile(acctal)
            if err != nil {
              fmt.Fprintln(os.Stderr, err)
              os.Exit(1)
            }
            fmt.Println("flag: " + string(flag))

            switch rules[1] {
            case "d":
              rint,_ := strconv.Atoi(rules[2])
              fint,_ := strconv.Atoi(string(flag))
              if rint > fint {
                fmt.Println("Now Large!!!: " + rules[2])
                for i, acnt := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(string(cuff), -1) {
                  fmt.Println("alert count",i+1, ": ", acnt)
                  if (len(acnt) > 1) {
                    acntb := strings.Split(acnt, "\t")
                    if strings.Index(acntb[0],rules[0]) != -1 {
                      ecnt,_ := strconv.Atoi(acntb[1])
                      if ecnt < 1 {
                        status += rules[0] + "\tERROR\t" + strconv.Itoa(rint) + "\n"
                        aacnt += fmt.Sprintf("%s\t%s\n", rules[0],rules[3])
                        acs := string(execmd("cp -f /home/" + acct + "/result/0_" + rules[0] + "_* /home/" + acct + "/out | tee && echo OK"))
                        if strings.Index(acs,"OK") != -1 {             
                          fmt.Println("Result Exits! Copy To:" + "/home/" + acct + "/out/" + rules[0])
                        }
                      } else {
                        ecnt = ecnt - 1
                        status += rules[0] + "\tWARNING\t" + strconv.Itoa(rint) + "\t" + strconv.Itoa(ecnt) + "\n"
                        aacnt += fmt.Sprintf("%s\t%s\n", rules[0],strconv.Itoa(ecnt))
                        fmt.Println(ecnt)
                        fmt.Println(rules[0] + "\t" + strconv.Itoa(ecnt))
                        acs := string(execmd("cp -f /home/" + acct + "/action/0_" + rules[0] + "_* /home/" + acct + "/out | tee && echo OK"))
                        if strings.Index(acs,"OK") != -1 {             
                          fmt.Println("Action Exits! Copy To:" + "/home/" + acct + "/out/" + rules[0])
                          fmt.Println(ecnt)
                        }
                      }
                    }
                  }
                }
              } else {
                status += rules[0] + "\tNORMAL\t" + strconv.Itoa(rint) + "\n"
                aacnt += fmt.Sprintf("%s\t%s\n", rules[0],rules[3])
                execmd("rm -f /home/" + acct + "/out/0_* ; rm -f /home/" + acct + "/out/_stay_")
              }
            case "D":
              rint,_ := strconv.Atoi(rules[2])
              fint,_ := strconv.Atoi(string(flag))
              if rint < fint {
                fmt.Println("Now Small!!!: " + rules[2])
                for i, acnt := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(string(cuff), -1) {
                  fmt.Println("alert count",i+1, ": ", acnt)
                  if (len(acnt) > 1) {
                    acntb := strings.Split(acnt, "\t")
                    if strings.Index(acntb[0],rules[0]) != -1 {
                      ecnt,_ := strconv.Atoi(acntb[1])
                      if ecnt < 1 {
                        status += rules[0] + "\tERROR\t" + strconv.Itoa(rint) + "\n"
                        aacnt += fmt.Sprintf("%s\t%s\n", rules[0],rules[3])
                        acs := string(execmd("cp -f /home/" + acct + "/result/0_" + rules[0] + "_* /home/" + acct + "/out | tee && echo OK"))
                        if strings.Index(acs,"OK") != -1 {
                          fmt.Println("Result Exits! Copy To:" + "/home/" + acct + "/out/" + rules[0])
                        }
                      } else {
                        ecnt = ecnt - 1
                        status += rules[0] + "\tWARNING\t" + strconv.Itoa(rint) + "\t" + strconv.Itoa(ecnt) + "\n"
                        aacnt += fmt.Sprintf("%s\t%s\n", rules[0],strconv.Itoa(ecnt))
                        fmt.Println(ecnt)
                        fmt.Println(rules[0] + "\t" + strconv.Itoa(ecnt))
                        acs := string(execmd("cp -f /home/" + acct + "/action/0_" + rules[0] + "_* /home/" + acct + "/out | tee && echo OK"))
                        if strings.Index(acs,"OK") != -1 {
                          fmt.Println("Action Exits! Copy To:" + "/home/" + acct + "/out/" + rules[0])
                          fmt.Println(ecnt)
                        }
                      }
                    }
                  }
                }
              }  else {
                status += rules[0] + "\tNORMAL\t" + strconv.Itoa(rint) + "\n"
                aacnt += fmt.Sprintf("%s\t%s\n", rules[0],rules[3])
                execmd("rm -f /home/" + acct + "/out/0_* ; rm -f /home/" + acct + "/out/_stay_")
              }
            case "s":
              r, _ := regexp.Compile(".*" + rules[2] + ".*")
              if (r.FindStringIndex(string(flag))) != nil {
                fmt.Println("Match!!!: " + rules[2])
                for i, acnt := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(string(cuff), -1) {
                  fmt.Println("alert count",i+1, ": ", acnt)
                  if (len(acnt) > 1) {
                    acntb := strings.Split(acnt, "\t")
                    if strings.Index(acntb[0],rules[0]) != -1 {
                      ecnt,_ := strconv.Atoi(acntb[1])
                      if ecnt < 1 {
                        status += rules[0] + "\tERROR\t" + rules[2] + "\n"
                        aacnt += fmt.Sprintf("%s\t%s\n", rules[0],rules[3])
                        acs := string(execmd("cp -f /home/" + acct + "/result/0_" + rules[0] + "_* /home/" + acct + "/out | tee && echo OK"))
                        if strings.Index(acs,"OK") != -1 {
                          fmt.Println("Result Exits! Copy To:" + "/home/" + acct + "/out/" + rules[0])
                        }
                      } else {
                        ecnt = ecnt - 1
                        status += rules[0] + "\tWARNING\t" + rules[2] + "\t" + strconv.Itoa(ecnt) + "\n"
                        aacnt += fmt.Sprintf("%s\t%s\n", rules[0],strconv.Itoa(ecnt))
                        fmt.Println(ecnt)
                        fmt.Println(rules[0] + "\t" + strconv.Itoa(ecnt))
                        acs := string(execmd("cp -f /home/" + acct + "/action/0_" + rules[0] + "_* /home/" + acct + "/out | tee && echo OK"))
                        if strings.Index(acs,"OK") != -1 {
                          fmt.Println("Action Exits! Copy To:" + "/home/" + acct + "/out/" + rules[0])
                          fmt.Println(ecnt)
                        }
                      }
                    }
                  }
                }
              } else {
                status += rules[0] + "\tNORMAL\t" + rules[2] + "\n"
                aacnt += fmt.Sprintf("%s\t%s\n", rules[0],rules[3])
                execmd("rm -f /home/" + acct + "/out/0_* ; rm -f /home/" + acct + "/out/_stay_")
              }
            case "S":
              r, _ := regexp.Compile(".*" + rules[2] + ".*")
              if (r.FindStringIndex(string(flag))) == nil {
                fmt.Println("Not Match!!!: " + rules[2])
                for i, acnt := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(string(cuff), -1) {
                  fmt.Println("alert count",i+1, ": ", acnt)
                  if (len(acnt) > 1) {
                    acntb := strings.Split(acnt, "\t")
                    if strings.Index(acntb[0],rules[0]) != -1 {
                      ecnt,_ := strconv.Atoi(acntb[1])
                      if ecnt < 1 {
                        status += rules[0] + "\tERROR\t" + rules[2] + "\n"
                        aacnt += fmt.Sprintf("%s\t%s\n", rules[0],rules[3])
                        acs := string(execmd("cp -f /home/" + acct + "/result/0_" + rules[0] + "_* /home/" + acct + "/out | tee && echo OK"))
                        if strings.Index(acs,"OK") != -1 {
                          fmt.Println("Result Exits! Copy To:" + "/home/" + acct + "/out/" + rules[0])
                        }
                      } else {
                        ecnt = ecnt - 1
                        status += rules[0] + "\tWARNING\t" + rules[2] + "\t" + strconv.Itoa(ecnt) + "\n"
                        aacnt += fmt.Sprintf("%s\t%s\n", rules[0],strconv.Itoa(ecnt))
                        fmt.Println(ecnt)
                        fmt.Println(rules[0] + "\t" + strconv.Itoa(ecnt))
                        acs := string(execmd("cp -f /home/" + acct + "/action/0_" + rules[0] + "_* /home/" + acct + "/out | tee && echo OK"))
                        if strings.Index(acs,"OK") != -1 {
                          fmt.Println("Action Exits! Copy To:" + "/home/" + acct + "/out/" + rules[0])
                          fmt.Println(ecnt)
                        }
                      }
                    }
                  }
                }
              } else {
                status += rules[0] + "\tNORMAL\t" + rules[2] + "\n"
                aacnt += fmt.Sprintf("%s\t%s\n", rules[0],rules[3])
                execmd("rm -f /home/" + acct + "/out/0_* ; rm -f /home/" + acct + "/out/_stay_")
              }
            }
            //default: 
            //}
            //execmd("rm -f " + acctal)
            execmd("mkdir -p /home/" + acct + "/metric/" + rules[0])
            execmd("mv -f " + acctal + " /home/" + acct + "/metric/" + rules[0])
          } else {
            for _, zcnt := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(string(cuff), -1) {
              zcntb := strings.Split(zcnt, "\t") 
              if strings.Index(zcntb[0],rules[0]) != -1 {
                aacnt += fmt.Sprintf("%s\t%s\n", rules[0],zcntb[1])
              }
            }
          }
        }
      }
      fmt.Println("/home/" + acct + "/alertcount")
      fmt.Println(aacnt)
      // ファイル書き出し
      execmd("rm -f /home/" + acct + "/alertcount")
      ioutil.WriteFile("/home/" + acct + "/alertcount", []byte(aacnt), os.ModePerm)
      execmd("rm -f /home/" + acct + "status_*")
      ioutil.WriteFile("/home/" + acct + "/status_" + string(statusout), []byte(status), os.ModePerm)

      execmd("rm -f /home/" + acct + "/out/_lock_")
    } else {
      fmt.Println(" not exits rule file, this account is disable.")
      if len(acct) > 1 {
        execmd("rm -f /home/" + acct + "status_*")
        ioutil.WriteFile("/home/" + acct + "/status_" + string(statusout), []byte(status), os.ModePerm)
      }
    }
  }
}

func execmd(command string) []byte {
  out, err := exec.Command(os.Getenv("SHELL"), "-c", command + " 2>&1 | tee").Output()
  if err != nil {
    log.Fatal(err)
  }
  //fmt.Printf("local exec command: %s\n", out)
  return out
}

func Exists(name string) bool {
    _, err := os.Stat(name)
    return !os.IsNotExist(err)
}
