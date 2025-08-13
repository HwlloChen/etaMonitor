package services

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// ServerType 服务器类型枚举
type ServerType int

const (
	Unknown ServerType = iota
	JavaEdition
	BedrockEdition
)

func (st ServerType) String() string {
	switch st {
	case JavaEdition:
		return "java"
	case BedrockEdition:
		return "bedrock"
	default:
		return "unknown"
	}
}

// MinecraftServer 统一的服务器信息结构
type MinecraftServer struct {
	ServerType    ServerType  `json:"server_type"`
	Version       VersionInfo `json:"version"`
	Players       Players     `json:"players"`
	Description   Description `json:"description"`
	Favicon       string      `json:"favicon,omitempty"`
	Ping          int         `json:"ping"`
	Online        bool        `json:"online"`
	RawData       interface{} `json:"raw_data,omitempty"`
}

// VersionInfo 版本信息结构体
type VersionInfo struct {
	Name     string `json:"name"`
	Protocol int    `json:"protocol"`
}

// Players 玩家信息结构体
type Players struct {
	Online int          `json:"online"`
	Max    int          `json:"max"`
	Sample []PlayerInfo `json:"sample,omitempty"`
}

// PlayerInfo 单个玩家信息
type PlayerInfo struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// Description 服务器描述信息
type Description struct {
	Text  string `json:"text,omitempty"`
	Extra []struct {
		Text  string `json:"text"`
		Color string `json:"color,omitempty"`
	} `json:"extra,omitempty"`
}

// UnmarshalJSON 自定义JSON解析，处理description字段可能是字符串或对象的情况
func (d *Description) UnmarshalJSON(data []byte) error {
	// 尝试解析为字符串
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		d.Text = str
		return nil
	}
	
	// 尝试解析为对象
	type Alias Description
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(d),
	}
	return json.Unmarshal(data, &aux)
}

// JavaServerStatus Java版服务器状态结构
type JavaServerStatus struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
		Sample []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"sample,omitempty"`
	} `json:"players"`
	Description interface{} `json:"description"`
	Favicon     string      `json:"favicon,omitempty"`
}

// BedrockServerStatus 基岩版服务器状态结构
type BedrockServerStatus struct {
	MOTD          string `json:"motd"`
	GameType      string `json:"game_type"`
	Map           string `json:"map"`
	PlayersOnline int    `json:"players_online"`
	MaxPlayers    int    `json:"max_players"`
	ServerID      string `json:"server_id"`
	GameMode      string `json:"game_mode"`
	GameModeNum   int    `json:"game_mode_num"`
	PortIPv4      int    `json:"port_ipv4"`
	PortIPv6      int    `json:"port_ipv6"`
	Version       string `json:"version"`
}

// DetectServerType 检测服务器类型
func DetectServerType(host string, javaPort, bedrockPort int) ServerType {
	// 首先尝试Java版
	if isJavaServer(host, javaPort) {
		return JavaEdition
	}
	
	// 然后尝试基岩版
	if isBedrockServer(host, bedrockPort) {
		return BedrockEdition
	}
	
	return Unknown
}

// isJavaServer 检测是否为Java版服务器
func isJavaServer(host string, port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 10*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	
	// 设置读写超时
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	
	// 尝试发送握手包
	handshake := createHandshakePacket(host, port)
	if _, err = conn.Write(handshake); err != nil {
		return false
	}
	
	// 发送状态请求
	statusRequest := createStatusRequestPacket()
	if _, err = conn.Write(statusRequest); err != nil {
		return false
	}
	
	// 尝试读取响应 - 只需要确认能读到数据
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil || n == 0 {
		return false
	}
	
	// 检查是否看起来像有效的Minecraft响应
	// 简单检查：数据应该包含JSON结构的开始
	responseStr := string(buffer[:n])
	return len(responseStr) > 10 // 最基本的长度检查
}

// isBedrockServer 检测是否为基岩版服务器
func isBedrockServer(host string, port int) bool {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return false
	}
	
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return false
	}
	defer conn.Close()
	
	// 发送Unconnected Ping
	ping := createUnconnectedPing()
	_, err = conn.Write(ping)
	if err != nil {
		return false
	}
	
	// 尝试读取响应
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Read(buffer)
	return err == nil
}

// JavaServerPing 查询Java版服务器状态
func JavaServerPing(host string, port int) (*MinecraftServer, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %v", err)
	}
	defer conn.Close()
	
	startTime := time.Now()
	
	// 设置读写超时
	conn.SetDeadline(time.Now().Add(15 * time.Second))
	
	// 发送握手包
	handshake := createHandshakePacket(host, port)
	if _, err = conn.Write(handshake); err != nil {
		return nil, fmt.Errorf("发送握手包失败: %v", err)
	}
	
	// 发送状态请求
	statusRequest := createStatusRequestPacket()
	if _, err = conn.Write(statusRequest); err != nil {
		return nil, fmt.Errorf("发送状态请求失败: %v", err)
	}
	
	// 使用更健壮的响应读取方法
	buffer := make([]byte, 65536) // 增加缓冲区大小
	totalRead := 0
	
	// 设置较长的初始超时
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	
	// 先读取包头信息
	headerBuffer := make([]byte, 16) // 足够读取包长度和包ID
	headerRead, err := conn.Read(headerBuffer)
	if err != nil {
		return nil, fmt.Errorf("读取包头失败: %v", err)
	}
	
	copy(buffer, headerBuffer[:headerRead])
	totalRead = headerRead
	
	// 解析包长度
	reader := bytes.NewReader(buffer[:totalRead])
	packetLen, err := readVarIntFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("读取包长度失败: %v", err)
	}
	
	// 计算需要的总字节数 (包长度 + VarInt编码的包长度字节数)
	packetLenBytes := getVarIntSize(packetLen)
	totalNeeded := int(packetLen) + packetLenBytes
	
	// 如果缓冲区还没有完整包，继续读取
	for totalRead < totalNeeded && totalRead < len(buffer) {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err := conn.Read(buffer[totalRead:])
		if err != nil {
			if totalRead >= 10 { // 如果有基本的数据，尝试解析
				break
			}
			return nil, fmt.Errorf("读取响应数据失败: %v", err)
		}
		totalRead += n
		
		// 避免读取过多数据
		if totalRead >= totalNeeded {
			break
		}
	}
	
	if totalRead == 0 {
		return nil, fmt.Errorf("未读取到任何响应数据")
	}
	
	// 重新解析响应
	reader = bytes.NewReader(buffer[:totalRead])
	
	// 读取包长度 (跳过，我们已经知道了)
	_, err = readVarIntFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("读取包长度失败: %v", err)
	}
	
	// 读取包ID
	packetID, err := readVarIntFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("读取包ID失败: %v", err)
	}
	
	if packetID != 0x00 {
		return nil, fmt.Errorf("无效的包ID: %d", packetID)
	}
	
	// 读取JSON长度
	jsonLength, err := readVarIntFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("读取JSON长度失败: %v", err)
	}
	
	// 计算剩余可读字节数
	remainingBytes := reader.Len()
	actualJsonLength := minInt(int(jsonLength), remainingBytes)
	
	// 读取实际可用的JSON数据
	jsonData := make([]byte, actualJsonLength)
	n, err := reader.Read(jsonData)
	if err != nil {
		return nil, fmt.Errorf("读取JSON数据失败: %v", err)
	}
	
	if n < actualJsonLength {
		jsonData = jsonData[:n] // 截断到实际读取的长度
	}
	
	// 尝试解析JSON，如果数据不完整，尝试修复
	var javaStatus JavaServerStatus
	if err = json.Unmarshal(jsonData, &javaStatus); err != nil {
		// 如果JSON不完整，尝试找到完整的部分
		jsonStr := string(jsonData)
		
		// 尝试找到最后一个完整的JSON结构
		lastBraceIndex := strings.LastIndex(jsonStr, "}")
		if lastBraceIndex > 0 {
			truncatedJson := jsonStr[:lastBraceIndex+1]
			if err = json.Unmarshal([]byte(truncatedJson), &javaStatus); err != nil {
				return nil, fmt.Errorf("解析JSON失败: %v (尝试修复后的数据: %s)", err, truncatedJson[:minInt(200, len(truncatedJson))])
			}
		} else {
			return nil, fmt.Errorf("解析JSON失败: %v (原始数据前200字符: %s)", err, jsonStr[:minInt(200, len(jsonStr))])
		}
	}
	
	// 转换为统一格式
	server := &MinecraftServer{
		ServerType: JavaEdition,
		Version: VersionInfo{
			Name:     javaStatus.Version.Name,
			Protocol: javaStatus.Version.Protocol,
		},
		Players: Players{
			Online: javaStatus.Players.Online,
			Max:    javaStatus.Players.Max,
		},
		Ping:    int(time.Since(startTime).Milliseconds()),
		Online:  true,
		RawData: javaStatus,
	}
	
	// 转换玩家样本
	for _, player := range javaStatus.Players.Sample {
		server.Players.Sample = append(server.Players.Sample, PlayerInfo{
			Name: player.Name,
			ID:   player.ID,
		})
	}
	
	// 处理描述字段
	if desc, ok := javaStatus.Description.(string); ok {
		server.Description.Text = desc
	} else if descObj, ok := javaStatus.Description.(map[string]interface{}); ok && descObj != nil {
		if text, exists := descObj["text"]; exists {
			server.Description.Text = fmt.Sprintf("%v", text)
		}
	}
	
	server.Favicon = javaStatus.Favicon
	
	return server, nil
}

// BedrockServerPing 查询基岩版服务器状态
func BedrockServerPing(host string, port int) (*MinecraftServer, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, fmt.Errorf("解析地址失败: %v", err)
	}
	
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %v", err)
	}
	defer conn.Close()
	
	startTime := time.Now()
	
	// 发送Unconnected Ping
	ping := createUnconnectedPing()
	if _, err = conn.Write(ping); err != nil {
		return nil, fmt.Errorf("发送ping失败: %v", err)
	}
	
	// 读取响应
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buffer := make([]byte, 2048)
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}
	
	// 解析响应
	bedrockStatus, err := parseBedrockResponse(buffer[:n])
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	
	// 转换为统一格式
	server := &MinecraftServer{
		ServerType: BedrockEdition,
		Version: VersionInfo{
			Name:     bedrockStatus.Version,
			Protocol: 0, // 基岩版协议版本号不固定
		},
		Players: Players{
			Online: bedrockStatus.PlayersOnline,
			Max:    bedrockStatus.MaxPlayers,
		},
		Description: Description{
			Text: bedrockStatus.MOTD,
		},
		Ping:    int(time.Since(startTime).Milliseconds()),
		Online:  true,
		RawData: bedrockStatus,
	}
	
	return server, nil
}

// AutoDetectServer 自动检测服务器类型并ping
func AutoDetectServer(host string, javaPort, bedrockPort int) (*MinecraftServer, string, error) {
	serverType := DetectServerType(host, javaPort, bedrockPort)
	
	switch serverType {
	case JavaEdition:
		server, err := JavaServerPing(host, javaPort)
		if err != nil {
			return nil, "", fmt.Errorf("Java版ping失败: %v", err)
		}
		return server, "java", nil
		
	case BedrockEdition:
		server, err := BedrockServerPing(host, bedrockPort)
		if err != nil {
			return nil, "", fmt.Errorf("基岩版ping失败: %v", err)
		}
		return server, "bedrock", nil
		
	default:
		return nil, "", fmt.Errorf("无法检测服务器类型")
	}
}

// createHandshakePacket 创建握手包
func createHandshakePacket(host string, port int) []byte {
	var buf bytes.Buffer
	
	// 包ID (0x00) - VarInt
	writeVarInt(&buf, 0x00)
	
	// 协议版本 (VarInt) - 使用更兼容的版本
	writeVarInt(&buf, 47) // 1.8 协议版本，兼容性更好
	
	// 服务器地址
	writeString(&buf, host)
	
	// 服务器端口
	binary.Write(&buf, binary.BigEndian, uint16(port))
	
	// 下一个状态 (1 = status)
	writeVarInt(&buf, 1)
	
	// 添加包长度前缀
	var final bytes.Buffer
	writeVarInt(&final, int32(buf.Len()))
	final.Write(buf.Bytes())
	
	return final.Bytes()
}

// createStatusRequestPacket 创建状态请求包
func createStatusRequestPacket() []byte {
	var buf bytes.Buffer
	
	// 包ID (0x00) - VarInt
	writeVarInt(&buf, 0x00)
	
	// 添加包长度前缀
	var final bytes.Buffer
	writeVarInt(&final, int32(buf.Len()))
	final.Write(buf.Bytes())
	
	return final.Bytes()
}

// createUnconnectedPing 创建Unconnected Ping包
func createUnconnectedPing() []byte {
	var buf bytes.Buffer
	
	// Packet ID
	buf.WriteByte(0x01)
	
	// Client timestamp
	binary.Write(&buf, binary.BigEndian, time.Now().Unix())
	
	// RakNet Magic
	magic := []byte{0x00, 0xFF, 0xFF, 0x00, 0xFE, 0xFE, 0xFE, 0xFE, 0xFD, 0xFD, 0xFD, 0xFD, 0x12, 0x34, 0x56, 0x78}
	buf.Write(magic)
	
	// Client GUID
	binary.Write(&buf, binary.BigEndian, int64(12345))
	
	return buf.Bytes()
}

// parseBedrockResponse 解析基岩版响应
func parseBedrockResponse(data []byte) (*BedrockServerStatus, error) {
	if len(data) < 35 {
		return nil, fmt.Errorf("响应数据太短")
	}
	
	// 检查包ID
	if data[0] != 0x1C {
		return nil, fmt.Errorf("无效的包ID: %d", data[0])
	}
	
	// 跳过时间戳和magic
	offset := 1 + 8 + 16 + 8 // PacketID + Timestamp + Magic + ServerGUID
	
	// 读取字符串长度
	if offset+2 > len(data) {
		return nil, fmt.Errorf("数据长度不足")
	}
	
	strLen := binary.BigEndian.Uint16(data[offset:])
	offset += 2
	
	// 读取服务器信息字符串
	if offset+int(strLen) > len(data) {
		return nil, fmt.Errorf("字符串长度超出数据范围")
	}
	
	serverInfo := string(data[offset : offset+int(strLen)])
	
	// 解析服务器信息
	parts := strings.Split(serverInfo, ";")
	if len(parts) < 6 {
		return nil, fmt.Errorf("服务器信息格式无效")
	}
	
	status := &BedrockServerStatus{}
	
	// 解析各个字段
	if len(parts) > 0 {
		status.MOTD = parts[0]
	}
	if len(parts) > 1 {
		status.GameType = parts[1]
	}
	if len(parts) > 2 {
		status.Map = parts[2]
	}
	if len(parts) > 3 {
		if online, err := strconv.Atoi(parts[3]); err == nil {
			status.PlayersOnline = online
		}
	}
	if len(parts) > 4 {
		if max, err := strconv.Atoi(parts[4]); err == nil {
			status.MaxPlayers = max
		}
	}
	if len(parts) > 5 {
		status.ServerID = parts[5]
	}
	if len(parts) > 6 {
		status.GameMode = parts[6]
	}
	if len(parts) > 7 {
		if mode, err := strconv.Atoi(parts[7]); err == nil {
			status.GameModeNum = mode
		}
	}
	if len(parts) > 8 {
		if port, err := strconv.Atoi(parts[8]); err == nil {
			status.PortIPv4 = port
		}
	}
	if len(parts) > 9 {
		if port, err := strconv.Atoi(parts[9]); err == nil {
			status.PortIPv6 = port
		}
	}
	if len(parts) > 10 {
		status.Version = parts[10]
	}
	
	return status, nil
}

// writeVarInt 写入VarInt
func writeVarInt(buf *bytes.Buffer, value int32) {
	for value >= 0x80 {
		buf.WriteByte(byte(value) | 0x80)
		value >>= 7
	}
	buf.WriteByte(byte(value))
}

// writeString 写入字符串
func writeString(buf *bytes.Buffer, s string) {
	writeVarInt(buf, int32(len(s)))
	buf.WriteString(s)
}

// readVarInt 读取VarInt
func readVarInt(conn net.Conn) (int32, error) {
	var result int32
	var shift uint
	
	for {
		b := make([]byte, 1)
		if _, err := conn.Read(b); err != nil {
			return 0, err
		}
		
		result |= int32(b[0]&0x7F) << shift
		
		if b[0]&0x80 == 0 {
			break
		}
		
		shift += 7
		if shift >= 32 {
			return 0, fmt.Errorf("VarInt太长")
		}
	}
	
	return result, nil
}

// readVarIntFromReader 从Reader读取VarInt
func readVarIntFromReader(reader *bytes.Reader) (int32, error) {
	var result int32
	var shift uint
	
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		
		result |= int32(b&0x7F) << shift
		
		if b&0x80 == 0 {
			break
		}
		
		shift += 7
		if shift >= 32 {
			return 0, fmt.Errorf("VarInt太长")
		}
	}
	
	return result, nil
}

// getVarIntSize 计算VarInt编码需要的字节数
func getVarIntSize(value int32) int {
	if value == 0 {
		return 1
	}
	
	size := 0
	for value > 0 {
		size++
		value >>= 7
	}
	return size
}

// minInt 返回两个整数中较小的一个
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}