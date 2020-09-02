package zero

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

// Conn wrap net.Conn
type Conn struct {
	sid        string
	rwConn     net.Conn
	sendCh     chan []byte
	done       chan error
	hbTimer    *time.Timer
	name       string
	msgChan    chan *Message
	hbInterval time.Duration
	hbTimeout  time.Duration
}

// GetName Get conn name
func (c *Conn) GetName() string {
	return c.name
}

// NewConn create new conn
func NewConn(c net.Conn, hbInterval time.Duration, hbTimeout time.Duration) *Conn {
	conn := &Conn{
		rwConn:     c,
		sendCh:     make(chan []byte, 100),
		done:       make(chan error),
		msgChan:    make(chan *Message, 100),
		hbInterval: hbInterval,
		hbTimeout:  hbTimeout,
		name:       c.RemoteAddr().String(),
		hbTimer:    time.NewTimer(hbInterval),
	}

	if conn.hbInterval == 0 {
		conn.hbTimer.Stop()
	}
	return conn
}

// Close close connection
func (c *Conn) Close() {
	c.hbTimer.Stop()
	c.rwConn.Close()
}

// SendMessage send message
func (c *Conn) SendMessage(msg *Message) error {
	pkg, err := Encode(msg)
	if err != nil {
		return err
	}

	c.sendCh <- pkg
	return nil
}

// writeCoroutine write coroutine
func (c *Conn) writeCoroutine(ctx context.Context) {
	hbData := make([]byte, 0)

	for {
		select {
		case <-ctx.Done():
			return

		case pkt := <-c.sendCh:

			if pkt == nil {
				continue
			}

			if _, err := c.rwConn.Write(pkt); err != nil {
				c.done <- err
			}

		case <-c.hbTimer.C:
			hbMessage := NewMessage(MsgHeartbeat, hbData)
			c.SendMessage(hbMessage)
			// 设置心跳timer
			if c.hbInterval > 0 {
				c.hbTimer.Reset(c.hbInterval)
			}
		}
	}
}

func (c *Conn) readCoroutine(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("err8")
			return

		default:

			// 设置超时
			if c.hbInterval > 0 {
				err := c.rwConn.SetReadDeadline(time.Now().Add(c.hbTimeout))
				if err != nil {
					fmt.Printf("err1=%s\n", err.Error())
					c.done <- err
					continue
				}
			}
			// 读取长度
			var msg Message

			buf := make([]byte, 12)
			_, err := io.ReadFull(c.rwConn, buf)
			if err != nil {
				fmt.Printf("err2=%s\n", err.Error())
				c.done <- err //关闭进程
				continue
			}
			bufReader := bytes.NewReader(buf)

			err = binary.Read(bufReader, binary.LittleEndian, &msg.tickTime)
			if err != nil {
				fmt.Printf("err3=%s\n", err.Error())
				c.done <- err
				continue
			}

			err = binary.Read(bufReader, binary.LittleEndian, &msg.msgID)
			if err != nil {
				fmt.Printf("err4=%s\n", err.Error())
				c.done <- err
				continue
			}

			err = binary.Read(bufReader, binary.LittleEndian, &msg.msgSize)
			if err != nil {
				fmt.Printf("err5=%s\n", err.Error())
				c.done <- err
				continue
			}

			err = binary.Read(bufReader, binary.LittleEndian, &msg.checksum)
			if err != nil {
				fmt.Printf("err6=%s\n", err.Error())
				c.done <- err
				continue
			}

			// 读取数据
			databuf := make([]byte, msg.msgSize)

			len5, err1 := io.ReadFull(c.rwConn, databuf)
			if err1 != nil {
				fmt.Printf("err7=%s\n", err.Error())
				c.done <- err1
				continue
			}
			msg.data = databuf
			// 解码
			//msgData, err := Decode(databuf)
			//if err != nil {
			//	c.done <- err
			//	continue
			//}

			//if msg.GetID() == MsgHeartbeat {
			//	continue
			//}
			fmt.Printf("_____msgid=%d,size=%d.len5=%d\n", int(msg.msgID), int(msg.msgSize), len5)

			//c.msgChan <- &msg
		}
	}
}
