package cast

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/buger/jsonparser"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"

	pb "github.com/vishen/go-chromecast/cast/proto"
)

const (
	dialerTimeout   = time.Second * 30
	dialerKeepAlive = time.Second * 30
)

var (
	// Global request id
	requestID int
)

type Connection struct {
	conn *tls.Conn

	resultChanMap map[int]chan *pb.CastMessage

	debug     bool
	connected bool
}

func NewConnection(debug bool) *Connection {
	c := &Connection{
		resultChanMap: map[int]chan *pb.CastMessage{},
		debug:         debug,
		connected:     false,
	}
	return c
}

func (c *Connection) Start(addr string, port int) error {
	if !c.connected {
		defer func() { go c.receiveLoop() }()
		return c.connect(addr, port)
	}
	return nil
}

func (c *Connection) SetDebug(debug bool) { c.debug = debug }

func (c *Connection) log(message string, args ...interface{}) {
	if c.debug {
		log.Printf("[connection] %s", fmt.Sprintf(message, args...))
	}
}

func (c *Connection) connect(addr string, port int) error {
	var err error
	dialer := &net.Dialer{
		Timeout:   dialerTimeout,
		KeepAlive: dialerKeepAlive,
	}
	c.conn, err = tls.DialWithDialer(dialer, "tcp", fmt.Sprintf("%s:%d", addr, port), &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return errors.Wrapf(err, "unable to connect to chromecast at '%s:%d'", addr, port)
	}
	c.connected = true
	return nil
}

func (c *Connection) SendAndWait(ctx context.Context, payload Payload, sourceID, destinationID, namespace string) (*pb.CastMessage, error) {

	if err := c.Send(payload, sourceID, destinationID, namespace); err != nil {
		return nil, err
	}

	// TODO(vishen): find better solution, super hacky, and it relying on
	// Send() to set the requestID. This is prone to race conditions!
	resultChan := make(chan *pb.CastMessage, 1)
	c.resultChanMap[requestID] = resultChan
	defer func() {
		delete(c.resultChanMap, requestID)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultChan:
		return result, nil
	}
}

func (c *Connection) Send(payload Payload, sourceID, destinationID, namespace string) error {
	// NOTE: Not concurrent safe, but currently only synchronous flow is possible
	// TODO(vishen): just make concurrent safe regardless of current flow
	requestID += 1
	payload.SetRequestId(requestID)

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "unable to marshal json payload")
	}
	payloadUtf8 := string(payloadJson)
	message := &pb.CastMessage{
		ProtocolVersion: pb.CastMessage_CASTV2_1_0.Enum(),
		SourceId:        &sourceID,
		DestinationId:   &destinationID,
		Namespace:       &namespace,
		PayloadType:     pb.CastMessage_STRING.Enum(),
		PayloadUtf8:     &payloadUtf8,
	}
	proto.SetDefaults(message)
	data, err := proto.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "unable to marshal proto payload")
	}

	c.log("%s -> %s [%s]: %s", sourceID, destinationID, namespace, payloadJson)

	if err := binary.Write(c.conn, binary.BigEndian, uint32(len(data))); err != nil {
		return errors.Wrap(err, "unable to write binary format")
	}
	if _, err := c.conn.Write(data); err != nil {
		return errors.Wrap(err, "unable to send data")
	}

	return nil
}

func (c *Connection) receiveLoop() {
	for {
		var length uint32
		if err := binary.Read(c.conn, binary.BigEndian, &length); err != nil {
			c.log("failed to binary read payload: %v", err)
			break
		}
		if length == 0 {
			c.log("empty payload received")
			continue
		}

		payload := make([]byte, length)
		i, err := io.ReadFull(c.conn, payload)
		if err != nil {
			c.log("failed to read payload: %v", err)
			continue
		}

		if i != int(length) {
			c.log("invalid payload, wanted: %d but read: %d", length, i)
			continue
		}

		message := &pb.CastMessage{}
		if err := proto.Unmarshal(payload, message); err != nil {
			c.log("failed to unmarshal proto cast message '%s': %v", payload, err)
			continue
		}

		c.log("%s <- %s [%s]: %s", *message.DestinationId, *message.SourceId, *message.Namespace, *message.PayloadUtf8)

		var headers PayloadHeader
		if err := json.Unmarshal([]byte(*message.PayloadUtf8), &headers); err != nil {
			c.log("failed to unmarshal proto message header: %v", err)
			continue
		}

		c.handleMessage(message, &headers)
	}
}

func (c *Connection) handleMessage(message *pb.CastMessage, headers *PayloadHeader) {

	messageType, err := jsonparser.GetString([]byte(*message.PayloadUtf8), "type")
	if err != nil {
		c.log("could not find 'type' key in response message %q: %s", *message.PayloadUtf8, err)
		return
	}

	switch messageType {
	case "PING":
		if err := c.Send(&PongHeader, *message.SourceId, *message.DestinationId, *message.Namespace); err != nil {
			c.log("unable to respond to 'PING': %v", err)
		}
	default:
		requestID, err := jsonparser.GetInt([]byte(*message.PayloadUtf8), "requestId")
		if err != nil {
			c.log("unable to find 'requestId' in proto payload '%s': %v", *message.PayloadUtf8, err)
		}
		if resultChan, ok := c.resultChanMap[int(requestID)]; ok {
			resultChan <- message
		}
	}
}
