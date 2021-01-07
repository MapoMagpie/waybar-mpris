module github.com/MapoMagpie/waybar-mpris

go 1.15

replace git.hrfee.pw/hrfee/waybar-mpris/mpris2client => ./mpris2client

require (
	git.hrfee.pw/hrfee/waybar-mpris/mpris2client v0.0.0-00010101000000-000000000000
	github.com/godbus/dbus/v5 v5.0.3
	github.com/hpcloud/tail v1.0.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/sys v0.0.0-20201116194326-cc9327a14d48 // indirect
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
)
