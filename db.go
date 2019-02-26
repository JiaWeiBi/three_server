package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/gofrontend/libgo/go/log"
	"time"
)

var mysqlDsn string
var mysqlConn *sql.DB
var StmpMap map[string]*sql.Stmt

func init() {
	mysqlDsn = fmt.Sprintf("%s:%s@%s(%s:%d)/%s?charset=utf8mb4", MysqlUser, MysqlPass, "tcp", MysqlHost, MysqlPort, "game")
	DB, err := sql.Open("mysql", mysqlDsn)
	if err != nil {
		fmt.Printf("Open mysql failed,err:%v\n", err)
		panic(err)
		return
	}
	DB.SetConnMaxLifetime(2 * time.Hour) //最大连接周期，超过时间的连接就close
	DB.SetMaxOpenConns(16)               //设置最大连接数
	DB.SetMaxIdleConns(4)                //设置闲置连接数
	mysqlConn = DB
	StmpMap = make(map[string]*sql.Stmt)
	if StmpMap["updateGold"],err=DB.Prepare("UPDATE threeUserInfo set gold=? where id=?"); err!=nil{
		log.Panicln("更新金币sql预编译失败", err)
		panic(err)
	}
	if StmpMap["updateScore"],err=DB.Prepare("UPDATE threeUserInfo set score=?,level=? where id=?"); err!=nil{
		log.Panicln("更新积分sql预编译失败", err)
		panic(err)
	}
}

func DBExit(){
	for _, stmt := range StmpMap{
		if err := stmt.Close();err != nil{
			log.Println("close stmt fail==", err)
		}
	}
	if err := mysqlConn.Close();err != nil{
		log.Println("close mysqlConn fail==", err)
	}
}