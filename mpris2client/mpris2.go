package mpris2client

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"
)

// Various paths and values to use elsewhere.
const (
	INTERFACE = "org.mpris.MediaPlayer2"
	PATH      = "/org/mpris/MediaPlayer2"
	// For the NameOwnerChanged signal.
	MATCH_NOC = "type='signal',path='/org/freedesktop/DBus',interface='org.freedesktop.DBus',member='NameOwnerChanged'"
	// For the PropertiesChanged signal. It doesn't match exactly (couldn't get that to work) so we check it manually.
	MATCH_PC = "type='signal',path='/org/mpris/MediaPlayer2',interface='org.freedesktop.DBus.Properties'"
	Refresh  = "refresh"
)

var knownPlayers = map[string]string{
	"plasma-browser-integration": "Browser",
	"noson":                      "Noson",
}

var knownBrowsers = map[string]string{
	"mozilla":  "Firefox",
	"chrome":   "Chrome",
	"chromium": "Chromium",
}

// Player represents an active media player.
type Player struct {
	Player                                            dbus.BusObject
	FullName, Name, Title, Artist, AlbumArtist, Album string
	Position                                          int64
	pid                                               uint32
	Playing, Stopped                                  bool
	metadata                                          map[string]dbus.Variant
	conn                                              *dbus.Conn
	poll                                              int
	interpolate                                       bool
}

// NewPlayer returns a new player object.
func NewPlayer(conn *dbus.Conn, name string, interpolate bool, poll int) (p *Player) {
	playerName := strings.ReplaceAll(name, INTERFACE+".", "")
	var pid uint32
	conn.BusObject().Call("org.freedesktop.DBus.GetConnectionUnixProcessID", 0, name).Store(&pid)
	for key, val := range knownPlayers {
		if strings.Contains(name, key) {
			playerName = val
			break
		}
	}
	if playerName == "Browser" {
		file, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err == nil {
			cmd := string(file)
			for key, val := range knownBrowsers {
				if strings.Contains(cmd, key) {
					playerName = val
					break
				}
			}
		}
	}
	p = &Player{
		Player:      conn.Object(name, PATH),
		conn:        conn,
		Name:        playerName,
		FullName:    name,
		pid:         pid,
		interpolate: interpolate,
		poll:        poll,
	}
	p.Refresh()
	return
}

func (p *Player) String() string {
	return fmt.Sprintf("Name: %s; Playing: %t; PID: %d", p.FullName, p.Playing, p.pid)
}

// Refresh grabs playback info.
func (p *Player) Refresh() (err error) {
	val, err := p.Player.GetProperty(INTERFACE + ".Player.PlaybackStatus")
	if err != nil {
		p.Playing = false
		p.Stopped = false
		p.metadata = map[string]dbus.Variant{}
		p.Title = ""
		p.Artist = ""
		p.AlbumArtist = ""
		p.Album = ""
		return
	}
	strVal := val.String()
	if strings.Contains(strVal, "Playing") {
		p.Playing = true
		p.Stopped = false
	} else if strings.Contains(strVal, "Paused") {
		p.Playing = false
		p.Stopped = false
	} else {
		p.Playing = false
		p.Stopped = true
	}
	metadata, err := p.Player.GetProperty(INTERFACE + ".Player.Metadata")
	if err != nil {
		p.metadata = map[string]dbus.Variant{}
		p.Title = ""
		p.Artist = ""
		p.AlbumArtist = ""
		p.Album = ""
		return
	}
	p.metadata = metadata.Value().(map[string]dbus.Variant)
	switch artist := p.metadata["xesam:artist"].Value().(type) {
	case []string:
		p.Artist = strings.Join(artist, ", ")
	case string:
		p.Artist = artist
	default:
		p.Artist = ""
	}
	switch albumArtist := p.metadata["xesam:albumArtist"].Value().(type) {
	case []string:
		p.AlbumArtist = strings.Join(albumArtist, ", ")
	case string:
		p.AlbumArtist = albumArtist
	default:
		p.AlbumArtist = ""
	}
	switch title := p.metadata["xesam:title"].Value().(type) {
	case string:
		p.Title = title
	default:
		p.Title = ""
	}
	switch album := p.metadata["xesam:album"].Value().(type) {
	case string:
		p.Album = album
	default:
		p.Album = ""
	}
	return nil
}

func µsToString(µs int64) string {
	seconds := int(µs / 1e6)
	minutes := int(seconds / 60)
	seconds -= minutes * 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// StringPosition figures out the track position in MM:SS/MM:SS, interpolating the value if necessary.
func (p *Player) StringPosition() string {
	// position is in microseconds so we prob need int64 to be safe
	v := p.metadata["mpris:length"].Value()
	var l int64
	if v != nil {
		l = v.(int64)
	} else {
		return ""
	}
	length := µsToString(l)
	if length == "" {
		return ""
	}
	pos, err := p.Player.GetProperty(INTERFACE + ".Player.Position")
	if err != nil {
		return ""
	}
	position := µsToString(pos.Value().(int64))
	if position == "" {
		return ""
	}
	if p.interpolate && position == µsToString(p.Position) {
		np := p.Position + int64(p.poll*1e6)
		position = µsToString(np)
	}
	p.Position = pos.Value().(int64)
	return position + "/" + length
}

// Next requests the next track.
func (p *Player) Next() { p.Player.Call(INTERFACE+".Player.Next", 0) }

// Previous requests the previous track.
func (p *Player) Previous() { p.Player.Call(INTERFACE+".Player.Previous", 0) }

// Toggle requests play/pause
func (p *Player) Toggle() { p.Player.Call(INTERFACE+".Player.PlayPause", 0) }

type Message struct {
	Name, Value string
}

type PlayerArray []*Player

func (ls PlayerArray) Len() int {
	return len(ls)
}

func (ls PlayerArray) Less(i, j int) bool {
	var states [2]uint8
	for i, p := range []bool{ls[i].Playing, ls[j].Playing} {
		if p {
			states[i] = 1
		}
	}
	// Reverse order
	return states[0] > states[1]
}

func (ls PlayerArray) Swap(i, j int) {
	ls[i], ls[j] = ls[j], ls[i]
}

type Mpris2 struct {
	List        PlayerArray
	Current     uint
	conn        *dbus.Conn
	Messages    chan Message
	interpolate bool
	poll        int
	autofocus   bool
}

func NewMpris2(conn *dbus.Conn, interpolate bool, poll int, autofocus bool) *Mpris2 {
	return &Mpris2{
		List:        PlayerArray{},
		Current:     0,
		conn:        conn,
		Messages:    make(chan Message),
		interpolate: interpolate,
		poll:        poll,
	}
}

// Listen should be run as a Goroutine. When players become available or are removed, an mpris2.Message is sent on mpris2.Mpris2.Messages with Name "add"/"remove" and Value as the player name. When a players state changes, a message is sent on mpris2.Mpris2.Messages with Name "refresh".
func (pl *Mpris2) Listen() {
	c := make(chan *dbus.Signal, 10)
	pl.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, MATCH_NOC)
	pl.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, MATCH_PC)
	pl.conn.Signal(c)
	for v := range c {
		if strings.Contains(v.Name, "NameOwnerChanged") {
			switch name := v.Body[0].(type) {
			case string:
				var pid uint32
				pl.conn.BusObject().Call("org.freedesktop.DBus.GetConnectionUnixProcessID", 0, name).Store(&pid)
				// Ignore playerctld again
				if strings.Contains(name, INTERFACE) && !strings.Contains(name, "playerctld") {
					if pid == 0 {
						pl.Remove(name)
						pl.Messages <- Message{Name: "remove", Value: name}
					} else {
						pl.New(name)
						pl.Messages <- Message{Name: "add", Value: name}
					}
				}
			}
		} else if strings.Contains(v.Name, "PropertiesChanged") && strings.Contains(v.Body[0].(string), INTERFACE+".Player") {
			pl.Refresh()
		}
	}
}

func (pl *Mpris2) Remove(fullName string) {
	currentName := pl.List[pl.Current].FullName
	var i int
	found := false
	for ind, p := range pl.List {
		if p.FullName == fullName {
			i = ind
			found = true
			break
		}
	}
	if !found {
		return
	}
	pl.List[0], pl.List[i] = pl.List[i], pl.List[0]
	pl.List = pl.List[1:]
	found = false
	for ind, p := range pl.List {
		if p.FullName == currentName {
			pl.Current = uint(ind)
			found = true
			break
		}
	}
	if !found {
		pl.Current = 0
		pl.Refresh()
		//fmt.Fprintln(WRITER, pl.JSON())
	}
}

func (pl *Mpris2) Reload() error {
	var buses []string
	err := pl.conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&buses)
	if err != nil {
		return err
	}
	for _, name := range buses {
		// Don't add playerctld, it just duplicates other players
		if strings.HasPrefix(name, INTERFACE) && !strings.Contains(name, "playerctld") {
			pl.New(name)
		}
	}
	return nil
}

func (pl *Mpris2) String() string {
	resp := ""
	pad := 0
	i := len(pl.List)
	for i != 0 {
		i /= 10
		pad++
	}
	for i, p := range pl.List {
		symbol := ""
		if uint(i) == pl.Current {
			symbol = "*"
		}
		resp += fmt.Sprintf("%0"+strconv.Itoa(pad)+"d", i) + symbol + ": " + p.String() + "\n"
	}
	return resp
}

func (pl *Mpris2) New(name string) {
	pl.List = append(pl.List, NewPlayer(pl.conn, name, pl.interpolate, pl.poll))
	if pl.autofocus {
		pl.Current = uint(len(pl.List) - 1)
	}
}

func (pl *Mpris2) Sort() {
	sort.Sort(pl.List)
	pl.Current = 0
}

func (pl *Mpris2) Refresh() {
	for i := range pl.List {
		pl.List[i].Refresh()
	}
	pl.Messages <- Message{Name: "refresh", Value: ""}
}
