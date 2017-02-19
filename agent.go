package main

import (
        "fmt"
	"log"
        "bytes"
	"golang.org/x/crypto/ssh"
        "github.com/pkg/sftp"
        "os"
        "os/exec"
        "regexp"
        "strings"
        "time"
        "flag"
        "strconv"
        "io/ioutil"
)

var LOCK=""
var PWD=""

func main() {
  //ip := "127.0.0.1"
  //port := "22"
  //user := "ady"
  _ip := flag.String("ip","","[-ip=IP Address] コントローラのIP")
  _port := flag.String("port","","[-port=Port Number] 接続ポート")
  _user := flag.String("user","","[-user=User Account] サーバーのユーザーアカウント")
  _key := flag.String("key","","[-key=(KEY FILE)] 鍵ファイルのパス")
  lptr := flag.Bool("l", false, "[-l] no lock mode **WARNING!**")
  _pwd := flag.String("pwd","","[-pwd=Directory] 実行ディレクトリ[default:/opt/sushi]")

  flag.Parse()

  ip := string(*_ip)
  port := string(*_port)
  user := string(*_user)
  key := string(*_key)
  pwdd := string(*_pwd)

  if *lptr == true {
    LOCK = "1"
    fmt.Println("No Lock Mode! **WARNING**")
  } else {
    LOCK = "0"
  }

  if len(pwdd) < 1 {
    PWD = "/opt/sushi"
  }

  if len(ip) < 8 {
    fmt.Println("Server IP is Empty.")
    os.Exit(1)
  }
  if len(port) < 2 {
    port = "22"
  }
  if len(user) < 2 {
    fmt.Println("User Acoount is Empty.")
    os.Exit(1)
  }
  if Exists(key) != true {
    fmt.Println("Key file is not Exists")
    os.Exit(1)
  }

  outs := string(execmd("ps -ef | grep \"agent \" | grep \"\\-ip\" | grep \"\\-token\" | grep \"\\-key\" | grep -v grep | wc | tr -s \" \" | cut -d \" \" -f 2 | tr -d \"\\n\""))
  pss,_ := strconv.Atoi(outs)
  if pss > 1 {
    fmt.Println("Agent Duplicating: " + outs)
    os.Exit(1)
  }

  cuff, err := ioutil.ReadFile(key)
  if err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }
  buf := string(cuff)

  fmt.Println(" -- list get -- ")
  //stayflag := 0
  acctlist := execssh(ip,port,user,buf,"ls -t ./out/*")
  if LOCK != "1" {
    if strings.Index(string(acctlist),"_lock_") != -1 {
      fmt.Println("Controller Working!! LOCKED")
      os.Exit(1)
    }
  }

  if strings.Index(string(acctlist),"_stay_") != -1 {
    fmt.Println(" -- stay rule -- ")
    dwfile(ip,port,user,buf,acctlist)
    acctlist = string(execmd("ls -t " + PWD + "/out/*"))
    acctlist = strings.Replace(acctlist,"./out/_stay_\n","",-1)
    //fmt.Println("\n -- \n " + acctlist + " \n -- \n ")
    //stayflag = 1
  } else {
    fmt.Println(" -- download get -- ")
    execmd("rm -f " + PWD + "/out/*")
    dwfile(ip,port,user,buf,acctlist)
  }
  fmt.Println(acctlist)
  r, _ := regexp.Compile(".*" + PWD + "/out/0_*")
  if (r.FindStringIndex(acctlist)) != nil {
    fmt.Println(" -- ACTION! -- ")
    actionexec(string(execmd("ls -t " + PWD + "/out/0_*")))
    acctlist = string(execmd("ls -t " + PWD + "/out/*"))
  }
  fmt.Println(" -- exec and upload -- ")
  fmt.Println(acctlist)
  execsh_up(ip,port,user,buf,acctlist)
  os.Exit(0)
}

func actionexec(actlists string) {
  for i, acct := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(actlists, -1) {
    if (len(acct) > 1) {
      fmt.Println("ACTION",i+1, ": ", acct)
      execmd(acct)
      if err := os.Remove(acct); err != nil {
        fmt.Println(err)
      }
    }
    //time.Sleep(1000)
  }
}

func execssh(ip,port,user,buf,command string) string {
  var stdoutBuf bytes.Buffer

  key, err := ssh.ParsePrivateKey([]byte(buf))
  if err != nil {
    panic(err)
  }

  config := &ssh.ClientConfig{
    User: user,
    Auth: []ssh.AuthMethod{
      ssh.PublicKeys(key),
    },
  }

  conn, err := ssh.Dial("tcp", ip+":"+port, config)
  if err != nil {
    log.Println(err)
  }
  defer conn.Close()

  session, err := conn.NewSession()
  if err != nil {
    log.Println(err)
  }
  defer session.Close()
  session.Stdout = &stdoutBuf
  session.Run("export PATH=$PATH ; " + command + " 2>&1 | tee")
  log.Println("ssh exec command: " + command)
  //fmt.Printf("-- \n%s\n-- \n", stdoutBuf)

  conn.Close()
  return stdoutBuf.String()
}

func Exists(name string) bool {
    _, err := os.Stat(name)
    return !os.IsNotExist(err)
}

func dwfile(ip,port,user,buf,srcfiles string) {
  key, err := ssh.ParsePrivateKey([]byte(buf))
  if err != nil {
    panic(err)
  }

  config := &ssh.ClientConfig{
    User: user,
    Auth: []ssh.AuthMethod{
      ssh.PublicKeys(key),
    },
  }

  client, err := ssh.Dial("tcp", ip+":"+port, config)
  if err != nil {
    log.Println(err)
  }
  defer client.Close()

  sftp, err := sftp.NewClient(client)
  if err != nil {
    log.Fatal(err)
  }
  defer sftp.Close()

  rep := regexp.MustCompile(".*repo_.*")
  for i, srcfile := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(srcfiles, -1) {
    if (len(srcfile) > 0) && strings.Index(srcfile,"_stay_") == -1 && rep.MatchString(srcfile) == false {
      fmt.Println("acct",i+1, ": ", srcfile)

      srcFile, err := sftp.Open(srcfile)
      if err != nil {
        log.Fatal(err)
      }
      defer srcFile.Close()

      dstFile, err := os.Create(PWD + "/" + srcfile)
      if err != nil {
        log.Fatal(err)
      } 
      defer dstFile.Close()

      srcFile.WriteTo(dstFile)

      execmd("chmod 700 " + PWD + "/" + srcfile)
      cksum := strings.Split(srcfile, "_")
      out := execmd("cksum " + srcfile + " | tr -s \" \" | cut -d \" \" -f 1 | tr -d \"\\n\"")
      cksums := ""
      if strings.Index(cksum[0],"0") != -1 {
        cksums = cksum[2]
      } else {
        cksums = cksum[1]
      }

      if cksums == string(out) {
        fmt.Println("cksum Match!: " + cksums + " "+ srcfile)

        session, err := client.NewSession()
        if err != nil {
          log.Println(err)
        }
        defer session.Close()
        //session.Run("export PATH=$PATH ; rm " + srcfile)
      } else {
        fmt.Println("cksum Miss!: " + cksums + " "+ srcfile)
        if err := os.Remove(srcfile); err != nil {
          fmt.Println(err)
        }
      }
      //time.Sleep(100)
    }
  }

  client.Close()
  sftp.Close()
}

func execmd(command string) []byte {
  out, err := exec.Command(os.Getenv("SHELL"), "-c", command + " 2>&1 | tee").Output()
  if err != nil {
    log.Fatal(err)
  }
  fmt.Printf("local exec command: %s\n", out)
  return out
}

func execsh_up(ip,port,user,buf,srcfiles string) {
  key, err := ssh.ParsePrivateKey([]byte(buf))
  if err != nil {
    panic(err)
  }

  config := &ssh.ClientConfig{
    User: user,
    Auth: []ssh.AuthMethod{
      ssh.PublicKeys(key),
    },
  }

  client, err := ssh.Dial("tcp", ip+":"+port, config)
  if err != nil {
    log.Println(err)
  }
  defer client.Close()

  sftp, err := sftp.NewClient(client)
  if err != nil {
    log.Fatal(err)
  }
  defer sftp.Close()

  t := time.Now()
  const layout = "2006-01-02-15-04-05"
  fmt.Println(t.Format(layout))

  rep := regexp.MustCompile(".*repo_.*")
  srcfiles = strings.Replace(srcfiles,".",PWD,-1)

  for i, srcfile := range regexp.MustCompile("\r\n|\n\r|\n|\r").Split(srcfiles, -1) {
    if (len(srcfile) > 0) && rep.MatchString(srcfile) == false {
      fmt.Println("acct",i+1, ": ", srcfile)
      out := execmd(srcfile)
      fmt.Println(string(out))

      nfile := strings.Split(srcfile, PWD + "/out/")
      cksum := strings.Split(nfile[1], "_")

      if cksum[0] == "0" {
        fmt.Println("Action/Result Remove: " + srcfile)
        execmd("rm -f " + srcfile)
      } else {
        file, err := sftp.Create("./in/" + cksum[0] + "_" + t.Format(layout))
        if err != nil {
          panic(err)
        }
        defer file.Close()

        _, err = file.Write(out)
        if err != nil {
          panic(err)
        }
      }
    }
    //time.Sleep(100)
  }

  client.Close()
  sftp.Close()
}
