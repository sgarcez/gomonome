package monome

import (
	"fmt"
	"net"

	"github.com/hypebeast/go-osc/osc"
)

type RingPressEvent struct {
	E int32
	S int32
}

func (c RingPressEvent) Type() string   { return "RingPress" }
func (c RingPressEvent) String() string { return fmt.Sprintf("%s: %d, %d", c.Type(), c.E, c.S) }

type Arc struct {
	bus    chan DeviceEvent
	client *osc.Client
	server *osc.Server
	conn   *net.PacketConn
	port   int32
	prefix string
	id     string
}

func StartArc(deviceport int32) (*Arc, error) {
	a, err := NewArc(deviceport)
	if err != nil {
		return nil, err
	}

	go func() {
		_ = a.ListenAndServe()
	}()

	a.Connect()

	return a, nil
}

func NewArc(deviceport int32) (*Arc, error) {
	a := Arc{}
	a.prefix = "monome"
	a.bus = make(chan DeviceEvent, 1)
	a.client = osc.NewClient("127.0.0.1", int(deviceport))

	conn, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		return &a, err
	}
	a.conn = &conn
	a.port = int32(conn.LocalAddr().(*net.UDPAddr).Port)

	a.server = &osc.Server{}
	a.server.Handle("/sys/id", func(msg *osc.Message) {
		a.id = msg.Arguments[0].(string)
		a.bus <- ReadyEvent{a.id}
	})
	a.server.Handle(fmt.Sprintf("/%s/enc/key", a.prefix), func(msg *osc.Message) {
		e := msg.Arguments[0].(int32)
		s := msg.Arguments[1].(int32)
		a.bus <- RingPressEvent{e, s}
	})

	return &a, err
}

func (a *Arc) ListenAndServe() error {
	err := a.server.Serve(*a.conn)
	if err != nil {
		netOpError, ok := err.(*net.OpError)
		if !ok || netOpError.Err.Error() != "use of closed network connection" {
			fmt.Println(err)
			return err
		}
	}
	return nil
}

func (a *Arc) Connect() {
	a.client.Send(osc.NewMessage("/sys/host", "127.0.0.1"))
	a.client.Send(osc.NewMessage("/sys/port", a.port))
	a.client.Send(osc.NewMessage("/sys/prefix", a.prefix))
	a.client.Send(osc.NewMessage("/sys/info/id"))
}

func (a *Arc) RingAll(enc int32, level int32) {
	a.client.Send(osc.NewMessage(fmt.Sprintf("/%s/ring/all", a.prefix), enc, level))
}

func (a *Arc) Read() DeviceEvent {
	return <-a.bus
}

func (a *Arc) Close() {
	if a.conn != nil {
		c := *a.conn
		c.Close()
		a.conn = nil
		close(a.bus)
	}
}
