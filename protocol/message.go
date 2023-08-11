package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"unsafe"
)

type (
	MessageType       byte
	MessageStatusType byte
	SerializeType     byte
)

const (
	magicNumber byte = 0x08
)

// 消息类型
const (
	Request MessageType = iota

	Response
)

// 消息状态类型
const (
	// normal request and response
	Normal MessageStatusType = iota

	// indicate some error happen
	Error
)

// 序列化类型
const (
	// raw []byte，不经过序列化/反序列化
	SerializeNone SerializeType = iota

	JSON

	ProtoBuffer
)

const (
	// contains error info of service invocation
	ServiceError = "__a_simple_rpc_error__"
)

var (
	MaxMessageLength = 0 // 最大消息长度，0:不限制长度

	// 消息太长
	ErrMessageTooLong = errors.New("message is too long")

	// metadata some key value missing
	ErrMetaKVMissing = errors.New("wrong metadata lines. some keys or values are missing")
)

type Message struct {
	*Header
	ServicePath   string
	ServiceMethod string
	Metadata      map[string]string
	Payload       []byte
	data          []byte
}

func NewMessage() *Message {
	header := Header{}
	header[0] = magicNumber

	return &Message{
		Header: &header,
	}
}

type Header [12]byte

// CheckMagicNumber checks whether header starts rpcx magic number.
func (h Header) CheckMagicNumber() bool {
	return h[0] == magicNumber
}

// Version returns version of rpcx protocol.
func (h Header) Version() byte {
	return h[1]
}

// SetVersion sets version for this header.
func (h *Header) SetVersion(v byte) {
	h[1] = v
}

// MessageType returns the message type.
func (h Header) MessageType() MessageType {
	return MessageType(h[2]&0x80) >> 7
}

// SetMessageType sets message type.
func (h *Header) SetMessageType(mt MessageType) {
	h[2] = h[2] | (byte(mt) << 7)
}

// IsHeartbeat returns whether the message is heartbeat message.
func (h Header) IsHeartbeat() bool {
	return h[2]&0x40 == 0x40
}

// SetHeartbeat sets the heartbeat flag.
func (h *Header) SetHeartbeat(hb bool) {
	if hb {
		h[2] = h[2] | 0x40
	} else {
		h[2] = h[2] &^ 0x40
	}
}

// IsOneway returns whether the message is one-way message.
// If true, server won't send responses.
func (h Header) IsOneway() bool {
	return h[2]&0x20 == 0x20
}

// SetOneway sets the oneway flag.
func (h *Header) SetOneway(oneway bool) {
	if oneway {
		h[2] = h[2] | 0x20
	} else {
		h[2] = h[2] &^ 0x20
	}
}

// MessageStatusType returns the message status type.
func (h Header) MessageStatusType() MessageStatusType {
	return MessageStatusType(h[2] & 0x03)
}

// SetMessageStatusType sets message status type.
func (h *Header) SetMessageStatusType(mt MessageStatusType) {
	h[2] = (h[2] &^ 0x03) | (byte(mt) & 0x03)
}

// SerializeType returns serialization type of payload.
func (h Header) SerializeType() SerializeType {
	return SerializeType((h[3] & 0xF0) >> 4)
}

// SetSerializeType sets the serialization type.
func (h *Header) SetSerializeType(st SerializeType) {
	h[3] = (h[3] &^ 0xF0) | (byte(st) << 4)
}

// Seq returns sequence number of messages.
func (h Header) Seq() uint64 {
	return binary.BigEndian.Uint64(h[4:])
}

// SetSeq sets  sequence number.
func (h *Header) SetSeq(seq uint64) {
	binary.BigEndian.PutUint64(h[4:], seq)
}

// 拷贝一个消息
func (m Message) Clone() *Message {
	header := *m.Header
	c := NewMessage()
	c.Header = &header
	c.ServicePath = m.ServicePath
	c.ServiceMethod = m.ServiceMethod
	return c
}

// 编码
func (m Message) Encode() []byte {
	// metadata
	var bb = bytes.NewBuffer(make([]byte, 0, len(m.Metadata)*64))
	encodeMetadata(m.Metadata, bb)
	meta := bb.Bytes()

	spL := len(m.ServicePath)
	smL := len(m.ServiceMethod)

	payload := m.Payload

	// spLen + sp + smLen + sm + metaL + meta + payloadLen + payload
	totalL := (4 + spL) + (4 + smL) + (4 + len(meta)) + (4 + len(payload))

	// header + dataLen + spLen + sp + smLen + sm + metaL + meta + payloadLen + payload
	metaStart := 12 + 4 + (4 + spL) + (4 + smL)

	payLoadStart := metaStart + (4 + len(meta))

	// header + dataLen + totalL
	l := 12 + 4 + totalL

	// todo 可以缓存小空间对象，重复使用，不必每次分配新的
	data := make([]byte, l)

	// header
	copy(data, m.Header[:])

	// total len
	binary.BigEndian.PutUint32(data[12:16], uint32(totalL))

	// service path
	binary.BigEndian.PutUint32(data[16:20], uint32(spL))
	copy(data[20:20+spL], StringToSliceByte(m.ServicePath))

	// service method
	binary.BigEndian.PutUint32(data[20+spL:24+spL], uint32(smL))
	copy(data[24+spL:metaStart], StringToSliceByte(m.ServiceMethod))

	// metadata
	binary.BigEndian.PutUint32(data[metaStart:metaStart+4], uint32(len(meta)))
	copy(data[metaStart+4:payLoadStart], meta)

	// payload
	binary.BigEndian.PutUint32(data[payLoadStart:payLoadStart+4], uint32(len(payload)))
	copy(data[payLoadStart+4:], payload)

	return data
}

// len,string,len,string,......
func encodeMetadata(m map[string]string, bb *bytes.Buffer) {
	if len(m) == 0 {
		return
	}
	d := make([]byte, 4)
	for k, v := range m {
		binary.BigEndian.PutUint32(d, uint32(len(k)))
		bb.Write(d)
		bb.Write(StringToSliceByte(k))
		binary.BigEndian.PutUint32(d, uint32(len(v)))
		bb.Write(d)
		bb.Write(StringToSliceByte(v))
	}
}

func decodeMetadata(l uint32, data []byte) (map[string]string, error) {
	m := make(map[string]string, 10)
	n := uint32(0)
	for n < l {
		// parse one key and value
		// key
		sl := binary.BigEndian.Uint32(data[n : n+4])
		n = n + 4
		if n+sl > l-4 {
			return m, ErrMetaKVMissing
		}
		k := string(data[n : n+sl])
		n = n + sl

		// value
		sl = binary.BigEndian.Uint32(data[n : n+4])
		n = n + 4
		if n+sl > l {
			return m, ErrMetaKVMissing
		}
		v := string(data[n : n+sl])
		n = n + sl
		m[k] = v
	}

	return m, nil
}

// 解码
func (m *Message) Decode(r io.Reader) error {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("panic in message decode: %v", err)
		}
	}()

	// header
	_, err := io.ReadFull(r, m.Header[:1])
	if err != nil {
		return err
	}
	if !m.Header.CheckMagicNumber() {
		return fmt.Errorf("wrong magic number: %v", m.Header[0])
	}
	_, err = io.ReadFull(r, m.Header[1:])
	if err != nil {
		return err
	}

	// total len
	lenData := make([]byte, 4)
	_, err = io.ReadFull(r, lenData)
	if err != nil {
		return err
	}
	l := binary.BigEndian.Uint32(lenData)
	totalLen := int(l)
	if MaxMessageLength > 0 && totalLen > MaxMessageLength {
		return ErrMessageTooLong
	}

	// 缓存全部数据
	if cap(m.data) >= totalLen {
		m.data = m.data[:totalLen]
	} else {
		m.data = make([]byte, totalLen)
	}
	data := m.data
	_, err = io.ReadFull(r, data)
	if err != nil {
		return err
	}

	n := 0
	// service path
	spL := binary.BigEndian.Uint32(data[n:4])
	n += 4
	nEnd := n + int(spL)
	m.ServicePath = SliceByteToString(data[n:nEnd])
	n = nEnd

	// service method
	sml := binary.BigEndian.Uint32(data[n : n+4])
	n += 4
	nEnd = n + int(sml)
	m.ServiceMethod = SliceByteToString(data[n:nEnd])
	n = nEnd

	// metadata
	mdl := binary.BigEndian.Uint32(data[n : n+4])
	n += 4
	nEnd = n + int(mdl)
	if mdl > 0 {
		m.Metadata, err = decodeMetadata(mdl, data[n:nEnd])
		if err != nil {
			return err
		}
	}
	n = nEnd

	// payload
	pyl := binary.BigEndian.Uint32(data[n : n+4])
	n += 4
	nEnd = n + int(pyl)
	m.Payload = data[n:nEnd]

	return nil
}

// 从字节流读取消息
func Read(r io.Reader) (m *Message, err error) {
	m = NewMessage()
	if err = m.Decode(r); err != nil {
		return nil, err
	}
	return m, nil
}

func SliceByteToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func StringToSliceByte(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}
