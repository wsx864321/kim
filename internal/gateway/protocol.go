package gateway

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

// 自定义二进制协议
// +--------+---------+---------+---------+------------------+
// | Magic  | Version | MsgType | Length  | Body (PB)        |
// | 2bytes | 1 byte  | 1 byte  | 4 bytes | N bytes          |
// +--------+---------+---------+---------+------------------+

type MsgType byte

const (
	MagicNumber uint16 = 0xABCD
	Version            = 1
	HeaderSize         = 8
	MaxBodySize        = 10 * 1024 * 1024 // 10MB，防止内存攻击
)

const (
	MsgTypeUnknown MsgType = iota
	MsgTypeLogin
	MsgTypeLogout
	MsgTypePing
	MsgTypePong
	MsgTypeUpstream // 上行消息（客户端→服务端）
	MsgTypePush     // 推送消息（服务端→客户端）
	MsgTypeACK      // 确认消息
)

var (
	ErrInvalidMagic       = errors.New("invalid magic number")
	ErrUnsupportedVersion = errors.New("unsupported protocol version")
	ErrBodyTooLarge       = errors.New("message body too large")
)

type Packet struct {
	MsgType MsgType
	Body    []byte
}

// EncodePacket 编码 Packet → 二进制字节流
func EncodePacket(p Packet) ([]byte, error) {
	bodyLen := len(p.Body)

	// 安全检查：限制消息体大小
	if bodyLen > MaxBodySize {
		return nil, fmt.Errorf("%w: %d bytes, max: %d bytes", ErrBodyTooLarge, bodyLen, MaxBodySize)
	}

	// 预分配完整缓冲区，避免多次扩容
	buf := make([]byte, HeaderSize+bodyLen)

	// 写入头部
	binary.BigEndian.PutUint16(buf[0:2], MagicNumber)
	buf[2] = Version
	buf[3] = byte(p.MsgType)
	binary.BigEndian.PutUint32(buf[4:8], uint32(bodyLen))

	// 写入 Body
	if bodyLen > 0 {
		copy(buf[HeaderSize:], p.Body)
	}

	return buf, nil
}

// DecodePacket 解码数据包（不设置超时，由调用方控制）
func DecodePacket(conn net.Conn) (*Packet, error) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}

	// 验证 Magic Number
	magic := binary.BigEndian.Uint16(header[0:2])
	if magic != MagicNumber {
		return nil, ErrInvalidMagic
	}

	// 验证 Version（支持版本兼容）
	version := header[2]
	if version != Version {
		return nil, fmt.Errorf("%w: got %d, expected %d", ErrUnsupportedVersion, version, Version)
	}

	msgType := MsgType(header[3])
	length := binary.BigEndian.Uint32(header[4:8])

	// 安全检查：限制消息体大小，防止内存攻击
	if length > MaxBodySize {
		return nil, fmt.Errorf("%w: %d bytes, max: %d bytes", ErrBodyTooLarge, length, MaxBodySize)
	}

	// 读取 body
	var body []byte
	if length > 0 {
		body = make([]byte, length)
		if _, err := io.ReadFull(conn, body); err != nil {
			return nil, fmt.Errorf("read body error: %w", err)
		}
	}

	return &Packet{
		MsgType: msgType,
		Body:    body,
	}, nil
}
