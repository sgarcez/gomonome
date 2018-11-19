package monome

import (
	"fmt"
	"net"

	"github.com/hypebeast/go-osc/osc"
)

type DeviceEvent interface {
	Type() string
}

type ReadyEvent struct {
	ID string
}

func (c ReadyEvent) Type() string   { return "Ready" }
func (c ReadyEvent) String() string { return fmt.Sprintf("%s: id: %s", c.Type(), c.ID) }

type KeyEvent struct {
	X int32
	Y int32
	S int32
}

func (c KeyEvent) Type() string   { return "Key" }
func (c KeyEvent) String() string { return fmt.Sprintf("%s: %d, %d, %d", c.Type(), c.X, c.Y, c.S) }

type TiltEvent struct {
	N int32
	X int32
	Y int32
	Z int32
}

func (c TiltEvent) Type() string { return "Tilt" }
func (c TiltEvent) String() string {
	return fmt.Sprintf("%s: %d, %d, %d, %d", c.Type(), c.N, c.X, c.Y, c.Z)
}

type Grid struct {
	bus    chan DeviceEvent
	client *osc.Client
	server *osc.Server
	conn   *net.PacketConn
	port   int32
	prefix string
	id     string
	width  int32
	height int32
}

func StartGrid(deviceport int32) (*Grid, error) {
	g, err := NewGrid(deviceport)
	if err != nil {
		return nil, err
	}

	go func() {
		_ = g.ListenAndServe()
	}()

	g.Connect()

	return g, nil
}

func NewGrid(deviceport int32) (*Grid, error) {
	g := Grid{}
	g.prefix = "monome"
	g.bus = make(chan DeviceEvent, 1)
	g.client = osc.NewClient("127.0.0.1", int(deviceport))

	conn, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		return &g, err
	}
	g.conn = &conn
	g.port = int32(conn.LocalAddr().(*net.UDPAddr).Port)

	g.server = &osc.Server{}
	g.server.Handle("/sys/id", func(msg *osc.Message) {
		g.id = msg.Arguments[0].(string)
		if g.width != 0 {
			g.bus <- ReadyEvent{g.id}
		}
	})
	g.server.Handle("/sys/size", func(msg *osc.Message) {
		g.width = msg.Arguments[0].(int32)
		g.height = msg.Arguments[1].(int32)
		if g.id != "" {
			g.bus <- ReadyEvent{g.id}
		}
	})

	g.server.Handle(fmt.Sprintf("/%s/grid/key", g.prefix), func(msg *osc.Message) {
		x := msg.Arguments[0].(int32)
		y := msg.Arguments[1].(int32)
		s := msg.Arguments[2].(int32)
		g.bus <- KeyEvent{x, y, s}
	})

	g.server.Handle(fmt.Sprintf("/%s/tilt", g.prefix), func(msg *osc.Message) {
		n := msg.Arguments[0].(int32)
		x := msg.Arguments[1].(int32)
		y := msg.Arguments[2].(int32)
		z := msg.Arguments[3].(int32)
		g.bus <- TiltEvent{n, x, y, z}
	})

	return &g, err
}

func (g *Grid) ListenAndServe() error {
	err := g.server.Serve(*g.conn)
	if err != nil {
		netOpError, ok := err.(*net.OpError)
		if !ok || netOpError.Err.Error() != "use of closed network connection" {
			fmt.Println(err)
			return err
		}
	}
	return nil
}

func (g *Grid) Connect() {
	g.client.Send(osc.NewMessage("/sys/host", "127.0.0.1"))
	g.client.Send(osc.NewMessage("/sys/port", g.port))
	g.client.Send(osc.NewMessage("/sys/prefix", g.prefix))
	g.client.Send(osc.NewMessage("/sys/info/id"))
	g.client.Send(osc.NewMessage("/sys/info/size"))
}

func (g *Grid) LedAll(v bool) {
	var state int32
	if v {
		state = 1
	}
	g.client.Send(osc.NewMessage(fmt.Sprintf("/%s/grid/led/all", g.prefix), state))
}

func (g *Grid) LedSet(x, y, s int32) {
	g.client.Send(osc.NewMessage(fmt.Sprintf("/%s/grid/led/set", g.prefix), x, y, s))
}

func (g *Grid) Read() DeviceEvent {
	return <-g.bus
}

func (g *Grid) Close() {
	g.LedAll(false)
	if g.conn != nil {
		c := *g.conn
		c.Close()
		g.conn = nil
		close(g.bus)
	}
}
