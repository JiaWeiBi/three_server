## [登录](#1)
## [匹配赛](#2)
* [开始匹配](#2-1)

* [取消匹配](#2-2)

* [匹配成功收到房间信息](#2-3)


## [好友赛](#3)
* [创建房间](#3-1)
* [进入房间](#3-2)
## [段位赛](#4)
## [房间内游戏逻辑](#5)
* [准备](#5-1)
* [退出](#5-2)


<h2 id="1">1.登录</h2>

### 路径 roomMgr.Login
### 登录消息结构

```
// 登录消息
	LoginMessage struct {
		Code     *string   `json:"code"`
		UserInfo *UserInfo `json:"userInfo"`
	}

    // 用户信息
	UserInfo struct {
		NickName  string `json:"nickName"`
		AvatarUrl string `json:"avatarUrl"`
		Gender    uint8  `json:"gender"`
		Province  string `json:"province"`
		City      string `json:"city"`
		Country   string `json:"country"`
	}
```
### 返回 用户信息结构

```
CastRole struct {
		Id   int64  `json:"id"`
		Name string `json:"name"`
		// 头像
		Icon string `json:"icon"`
		// 金币
		Gold int64 `json:"gold"`
		// 等级
		Level int `json:"level"`
		// 积分
		Score int `json:"score"`
		// 性别
		Gender uint8 `json:"gender"`
	}
```

<h2 id="2">1.匹配赛</h2>

<h3 id="2-1">开始匹配</h3>

### 路径 roomMgr.StartMatch

### 参数

```
// 匹配请求消息 type: 0-普通匹配 1-好友赛 2-段位赛
	HallMatchMessage struct {
		Type int `json:"type"` 
	}

	// 匹配成功后收到 "ok" 返回，失败无返回
```

<h3 id="2-2">取消匹配</h3>

#### 路径 roomMgr.CancelMatch

#### 参数 同匹配请求消息，type意义不变 好友赛不存在取消匹配 

#### 结果推送路径 onCancelMatch 值为1表示取消成功 0表示失败


<h3 id="2-3">匹配成功，收到房间信息</h3>

### 推送消息路径 onRoomInfo

### 消息结构

```
// 房间信息 用于推送
	CastRoomInfo struct {
		Id            int64          `json:"id"`
		// 房间状态
		//0:双方未准备 1:1号准备 2:2号准备
		// 11:FPlayer下子 12:FPlayer走子 13:FPlayer揪子 21:SPlayer下子 22:SPlayer走子 23:SPlayer揪子
		Status        uint8          `json:"status"`
		PointMap      map[string]int `json:"pointmap"`
		// 房间类型 0:普通匹配 1:好友赛 2;段位赛
		Type          int            `json:"type"`
		// 倍率
		Magnification int            `json:"mag"`
		// 求和人 0：没有人求和 1:一号求和 2:二号求和
		SuePeace   int    `json:"peace"`
		ActionTime int64 `json:"actiontime"`

		FPlayer       *CastRole `json:"fplayer"`
		SPlayer       *CastRole `json:"splayer"`
		FPlayerHelper *CastRole `json:"fhelper"`
		SPlayerHelper *CastRole `json:"shelper"`
	}
```

<h3 id="3-1">好友赛</h3>

#### 创建房间 同匹配赛 type为1

#### 创建成功会直接受到房间信息，同匹配赛， 房间类型为1

<h3 id="3-2">进入好友房间</h3>

#### 请求路径 roomMgr.EnterFriendRoom

#### 请求消息格式

```
// 进入好友房间 成功则收到房间信息
	HallEnterFriendRoom struct {
		// 好友id
		Fid    int64 `json:"fid"`

		// 房间id
		RoomId int64 `json:"roomid"`
	}
```

<h3 id="5-1">准备</h3>

### 请求路径 roomHandlers.Ready

### 消息结构 可为空

<h3 id="5-2">取消准备</h3>

### 请求路径 roomHandlers.CancleReady

### 消息结构 可为空

<h3 id="5-3">退出</h3>

### 注意，在对局或准备时不能强制退出，必须先认输或取消准备

### 请求路径 roomHandlers.Quit

### 消息结构 可为空