package monome

import (
	"fmt"
	"io"
	"net"
	"regexp"

	"github.com/hypebeast/go-osc/osc"
)

type ControlEventType int

func (s ControlEventType) String() string {
	return []string{"List", "Add", "Remove"}[s]
}

const (
	List = iota
	Add
	Remove
)

type ControlEvent struct {
	EventType  ControlEventType
	ID         string
	DeviceType string
	Port       int32
}

func (c ControlEvent) DeviceKind() string {
	match, _ := regexp.MatchString("monome arc*", c.DeviceType)
	if match {
		return "arc"
	}
	return "grid"
}

type SerialOSC struct {
	bus    chan ControlEvent
	client *osc.Client
	server *osc.Server
	conn   *net.PacketConn
	port   int32
}

func StartSerialOSC() (*SerialOSC, error) {
	s, err := NewSerialOSC()
	if err != nil {
		return nil, err
	}

	go func() {
		err := s.ListenAndServe()
		if err != nil {
			fmt.Printf("error from serial-osc server: %v", err)
		}
	}()

	s.List()

	return s, nil
}
func NewSerialOSC() (*SerialOSC, error) {
	s := SerialOSC{}
	s.bus = make(chan ControlEvent, 1)
	s.client = osc.NewClient("127.0.0.1", 12002)

	conn, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		return &s, err
	}
	s.conn = &conn
	s.port = int32(conn.LocalAddr().(*net.UDPAddr).Port)

	s.server = &osc.Server{}
	s.server.Handle("/serialosc/device", func(msg *osc.Message) {
		// osc.PrintMessage(msg)
		s.bus <- ControlEvent{List, msg.Arguments[0].(string), msg.Arguments[1].(string), msg.Arguments[2].(int32)}
	})
	s.server.Handle("/serialosc/add", func(msg *osc.Message) {
		// must always re-subscribe
		s.Subscribe()
		s.bus <- ControlEvent{Add, msg.Arguments[0].(string), msg.Arguments[1].(string), msg.Arguments[2].(int32)}
	})
	s.server.Handle("/serialosc/remove", func(msg *osc.Message) {
		// must always re-subscribe
		s.Subscribe()
		s.bus <- ControlEvent{Remove, msg.Arguments[0].(string), msg.Arguments[1].(string), msg.Arguments[2].(int32)}
	})

	return &s, err
}

func (s *SerialOSC) ListenAndServe() error {
	err := s.server.Serve(*s.conn)
	if err != nil {
		netOpError, ok := err.(*net.OpError)
		if !ok || netOpError.Err.Error() != "use of closed network connection" {
			fmt.Println(err)
			return err
		}
	}
	return nil
}

func (s *SerialOSC) List() {
	s.client.Send(osc.NewMessage("/serialosc/list", "127.0.0.1", s.port))
}

// Subscribe registers an interest in add/remove events from serial-osc.
func (s *SerialOSC) Subscribe() {
	s.client.Send(osc.NewMessage("/serialosc/notify", "127.0.0.1", s.port))
}

func (s *SerialOSC) Read() (ControlEvent, error) {
	if s.bus == nil {
		return ControlEvent{}, io.EOF
	}
	return <-s.bus, nil
}

func (s *SerialOSC) Close() {
	if s.conn != nil {
		c := *s.conn
		c.Close()
		close(s.bus)
	}
}
