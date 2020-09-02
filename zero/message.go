package zero

import (
	"fmt"
)

// Message struct
type MessageHeader struct {
	tickTime uint32
	msgID    uint16
	msgSize  uint16
	checksum [4]byte
}

type Message struct {
	tickTime uint32
	msgID    uint16
	msgSize  uint16
	checksum [4]byte
	data     []byte
}

// NewMessage create a new message
func NewMessage(msgID uint16, data []byte) *Message {
	msg := &Message{
		tickTime: uint32(0),
		msgID:    msgID,
		msgSize:  uint16(len(data)) + 12,
		checksum: [4]byte{'$', '3', '&', '@'},
		data:     data,
	}

	//msg.checksum = msg.calcChecksum()
	return msg
}

// GetData get message data
func (msg *Message) GetData() []byte {
	return msg.data
}

// GetID get message ID
func (msg *Message) GetID() uint16 {
	return msg.msgID
}

func (msg *Message) GetSize() uint16 {
	return msg.msgSize
}

// Verify verify checksum
func (msg *Message) Verify() bool {
	return [4]byte{'$', '3', '&', '@'} == msg.checksum
}

func (msg *Message) String() string {
	return fmt.Sprintf("Size=%d ID=%d DataLen=%d ", msg.msgSize, msg.GetID(), len(msg.GetData()))
}
