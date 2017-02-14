package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"strings"

	"github.com/spf13/viper"
	zif "github.com/zif/zif"
	data "github.com/zif/zif/data"

	log "github.com/sirupsen/logrus"
)

// these two are inserted by the makefile at build time
var (
	Version   = "N/A"
	BuildTime = "N/A"
)

func SetupLocalPeer(addr string) *zif.LocalPeer {
	var lp zif.LocalPeer

	if lp.ReadKey() != nil {
		lp.GenerateKey()
		lp.WriteKey()
	}
	lp.Setup()

	return &lp
}

func main() {

	log.SetLevel(log.DebugLevel)
	formatter := new(log.TextFormatter)
	formatter.FullTimestamp = true
	formatter.TimestampFormat = "15:04:05"
	log.SetFormatter(formatter)

	os.Mkdir("./data", 0777)

	SetupConfig()

	addr := viper.GetString("bind.zif")
	fmt.Println(addr)

	port, _ := strconv.Atoi(strings.Split(addr, ":")[1])

	lp := SetupLocalPeer(fmt.Sprintf("%s:%v", addr))
	lp.LoadEntry()

	log.WithFields(log.Fields{
		"version": Version,
		"built":   BuildTime,
	}).Info("Starting zifd")

	if viper.GetBool("tor.enabled") {
		_, onion, err := zif.SetupZifTorService(port, viper.GetInt("tor.control"),
			fmt.Sprintf("%s/cookie", viper.GetString("tor.cookiePath")))

		if err == nil {
			lp.PublicAddress = onion
			lp.Entry.PublicAddress = onion
			lp.SetSocks(true)
			lp.SetSocksPort(viper.GetInt("tor.socks"))
			lp.Peer.Streams().Socks = true
			lp.Peer.Streams().SocksPort = viper.GetInt("tor.socks")
		} else {
			panic(err)
		}

		// should this override tor?
	} else if viper.GetBool("socks.enabled") {
		lp.SetSocks(true)
		lp.SetSocksPort(viper.GetInt("socks.port"))
		lp.Peer.Streams().Socks = true
		lp.Peer.Streams().SocksPort = viper.GetInt("socks.port")

		// TODO: configurable public address
	} else {
		if lp.Entry.PublicAddress == "" {
			log.Debug("Local peer public address is nil, attempting to fetch")
			ip := zif.ExternalIp()
			log.Debug("External IP is ", ip)
			lp.Entry.PublicAddress = ip
		}
	}

	lp.Entry.Port = port
	lp.Entry.SetLocalPeer(lp)
	lp.SignEntry()
	lp.SaveEntry()

	err := lp.SaveEntry()

	if err != nil {
		panic(err)
	}

	lp.Database = data.NewDatabase(viper.GetString("database.path"))

	err = lp.Database.Connect()

	if err != nil {
		log.Fatal(err.Error())
	}

	lp.Listen(viper.GetString("bind.zif"))

	log.Info("My name: ", lp.Entry.Name)
	s, _ := lp.Address().String()
	log.Info("My address: ", s)

	commandServer := zif.NewCommandServer(lp)
	var httpServer zif.HttpServer
	httpServer.CommandServer = commandServer
	go httpServer.ListenHttp(viper.GetString("bind.http"))

	err = lp.StartExploring()

	if err != nil {
		log.Error(err.Error())
	}

	// Listen for SIGINT
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)

	for _ = range sigchan {
		lp.Close()

		os.Exit(0)
	}
}
