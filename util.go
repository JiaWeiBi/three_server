package main

import "github.com/lonng/nano/session"
import "fmt"
import "log"

// 通过id获取Role
var GetRoleById func (int64) (*Role, bool)

// 检查是否已登录
func CheckLogin(s *session.Session) bool {
	if s.UID() == 0{
		fmt.Println("=====未登录====")
		err := s.Push("notice", NoticeMessage{Content:"请先登录"})
		if err != nil{
			log.Println("提示失败==", err)
		}
		return false
	}
	return true
}