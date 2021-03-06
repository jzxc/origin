package network

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"

	"time"
)

//IWebsocketClient ...
type IWebsocketClient interface {
	Init(slf IWebsocketClient, strurl, strPath string, bproxy bool, timeoutsec time.Duration) error
	Start() error
	WriteMessage(msg []byte) error
	OnDisconnect() error
	OnConnected() error
	OnReadMessage(msg []byte) error
	ReConnect()
}

//WebsocketClient ...
type WebsocketClient struct {
	WsDailer   *websocket.Dialer
	conn       *websocket.Conn
	url        string
	state      int //0未连接状态   1正在重连   2连接状态
	bwritemsg  chan []byte
	slf        IWebsocketClient
	timeoutsec time.Duration

	bRun bool
}

//Init ...
func (ws *WebsocketClient) Init(slf IWebsocketClient, strurl, strPath string, bproxy bool, timeoutsec time.Duration) error {

	ws.timeoutsec = timeoutsec
	ws.slf = slf
	if bproxy == true {
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(strPath)
		}

		if timeoutsec > 0 {
			tosec := timeoutsec * time.Second
			ws.WsDailer = &websocket.Dialer{Proxy: proxy, HandshakeTimeout: tosec}
		} else {
			ws.WsDailer = &websocket.Dialer{Proxy: proxy}
		}
	} else {
		if timeoutsec > 0 {
			tosec := timeoutsec * time.Second
			ws.WsDailer = &websocket.Dialer{HandshakeTimeout: tosec}
		} else {
			ws.WsDailer = &websocket.Dialer{}
		}
	}

	ws.url = strurl
	ws.bwritemsg = make(chan []byte, 1000)

	return nil
}

//OnRun ...
func (ws *WebsocketClient) OnRun() error {
	for {
		if ws.bRun == false {
			break
		}

		if ws.state == 0 {
			time.Sleep(1 * time.Second)
			ws.StartConnect()
		} else if ws.state == 1 {
			log.Println("需要进行重连")
			ws.conn.Close()
			ws.state = 0
			ws.slf.OnDisconnect()
		} else if ws.state == 2 {
			ws.conn.SetReadDeadline(time.Now().Add(ws.timeoutsec * time.Second))
			_, message, err := ws.conn.ReadMessage()

			if err != nil {
				log.Printf("到服务器的连接断开 %+v\n", err)
				ws.conn.Close()
				ws.state = 0
				ws.slf.OnDisconnect()
				continue
			}

			ws.slf.OnReadMessage(message)
		}
	}

	return nil
}

//StartConnect ...
func (ws *WebsocketClient) StartConnect() error {

	var err error
	ws.conn, _, err = ws.WsDailer.Dial(ws.url, nil)
	fmt.Printf("connecting %s, %+v\n", ws.url, err)
	if err != nil {
		return err
	}

	ws.state = 2
	ws.slf.OnConnected()

	return nil
}

//Start ...
func (ws *WebsocketClient) Start() error {
	if ws.bRun == false {
		ws.bRun = true
		ws.state = 0
		go ws.OnRun()
		go ws.writeMsg()
	}
	return nil
}

//触发
func (ws *WebsocketClient) writeMsg() error {
	timerC := time.NewTicker(time.Second * 5).C
	for {
		if ws.bRun == false {
			break
		}

		if ws.state == 0 {
			time.Sleep(1 * time.Second)
			continue
		}
		select {
		case <-timerC:
			if ws.state == 2 {
				ws.WriteMessage([]byte(`ping`))
			}
		case msg := <-ws.bwritemsg:
			if ws.state == 2 {
				ws.conn.SetWriteDeadline(time.Now().Add(ws.timeoutsec * time.Second))
				err := ws.conn.WriteMessage(websocket.TextMessage, msg)

				if err != nil {
					fmt.Print(err)
					ws.state = 0
					ws.conn.Close()
					ws.slf.OnDisconnect()
				}
			}
		}
	}

	return nil
}

//ReConnect ...
func (ws *WebsocketClient) ReConnect() {
	ws.state = 1
}

//WriteMessage ...
func (ws *WebsocketClient) WriteMessage(msg []byte) error {
	ws.bwritemsg <- msg
	return nil
}

//OnDisconnect ...
func (ws *WebsocketClient) OnDisconnect() error {

	return nil
}

//OnConnected ...
func (ws *WebsocketClient) OnConnected() error {

	return nil
}

//OnReadMessage 触发
func (ws *WebsocketClient) OnReadMessage(msg []byte) error {

	return nil
}

//Stop ...
func (ws *WebsocketClient) Stop() {
	ws.bRun = false
}
